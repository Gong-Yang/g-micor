package pgsql

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"log/slog"
	"reflect"
	"strings"
	"time"

	"github.com/Gong-Yang/g-micor/syncx"
)

type DBEntity interface {
}

type fieldScanKind uint8

const (
	scanDirect fieldScanKind = iota
	scanJSON
	scanPtrString
	scanPtrBool
	scanPtrInt
	scanPtrUint
	scanPtrFloat
	scanPtrTime
)

type fieldMeta struct {
	Name         string
	DBName       string
	Index        int
	IsPrimaryKey bool
	GoType       reflect.Type
	ScanKind     fieldScanKind
	ElemType     reflect.Type // 仅用于 *基础类型
}

type Table[T DBEntity] struct {
	name         string
	pkField      *fieldMeta
	fields       []*fieldMeta
	insertFields []*fieldMeta // 不含 id
	selectOneSQL string
}

// ---- 分页结果 ----

type Page[T DBEntity] struct {
	Total int64 `json:"total"`
	Items []*T  `json:"items"`
}

// ---- 缓存相关 ----

var (
	tableStore  = syncx.NewResourceManager[any]()
	timeType    = reflect.TypeOf(time.Time{})
	valuerType  = reflect.TypeOf((*driver.Valuer)(nil)).Elem()
	scannerType = reflect.TypeOf((*sql.Scanner)(nil)).Elem()
)

func GetTable[T DBEntity](tableName string) *Table[T] {
	var t T
	tType := reflect.TypeOf(t)
	if tType == nil || tType.Kind() != reflect.Struct {
		panic("T must be a struct")
	}

	cacheKey := tType.PkgPath() + "." + tType.Name() + tableName

	tableObj, err := tableStore.GetResource(cacheKey, func() (any, error) {
		var (
			fields       []*fieldMeta
			insertFields []*fieldMeta
			columns      []string
			placeholders []string
			allColumns   []string
			pkField      *fieldMeta
		)

		for i := 0; i < tType.NumField(); i++ {
			field := tType.Field(i)

			// 过滤未导出字段
			if !field.IsExported() {
				continue
			}

			dbName := field.Tag.Get("db")
			if dbName == "-" {
				continue
			}
			if dbName == "" {
				dbName = field.Name
			}

			scanKind, elemType := classifyScanKind(field.Type)

			meta := &fieldMeta{
				Name:         field.Name,
				DBName:       dbName,
				Index:        i,
				IsPrimaryKey: dbName == "id",
				GoType:       field.Type,
				ScanKind:     scanKind,
				ElemType:     elemType,
			}

			allColumns = append(allColumns, dbName)
			fields = append(fields, meta)

			if meta.IsPrimaryKey {
				if field.Type.Kind() != reflect.Int64 {
					return nil, fmt.Errorf("db id must be int64")
				}
				pkField = meta
				continue
			}

			insertFields = append(insertFields, meta)
			columns = append(columns, dbName)
			placeholders = append(placeholders, fmt.Sprintf("$%d", len(columns)))
		}

		if pkField == nil {
			return nil, fmt.Errorf("table %s must have an id field", tType.Name())
		}

		return &Table[T]{
			name:         tableName,
			fields:       fields,
			insertFields: insertFields,
			pkField:      pkField,
			selectOneSQL: fmt.Sprintf(
				"SELECT %s FROM %s WHERE id = $1",
				strings.Join(allColumns, ", "),
				tableName,
			),
		}, nil
	})
	if err != nil {
		slog.ErrorContext(context.Background(), "get table error", "err", err)
		panic(err)
	}
	return tableObj.(*Table[T])
}
func (t *Table[T]) hasExplicitPK(entity *T) bool {
	v := reflect.ValueOf(entity).Elem().Field(t.pkField.Index)
	return !v.IsZero()
}

func (t *Table[T]) insertColumns(includePK bool) []*fieldMeta {
	if !includePK {
		return t.insertFields
	}
	cols := make([]*fieldMeta, 0, len(t.insertFields)+1)
	cols = append(cols, t.pkField)
	cols = append(cols, t.insertFields...)
	return cols
}

func (t *Table[T]) extractArgsByFields(ctx context.Context, entity *T, fields []*fieldMeta) ([]any, error) {
	val := reflect.ValueOf(entity).Elem()
	args := make([]any, 0, len(fields))

	for _, field := range fields {
		fieldValue := val.Field(field.Index)
		value, err := getValue(fieldValue)
		if err != nil {
			slog.ErrorContext(ctx, "get value error", "field", field.Name, "err", err)
			return nil, err
		}
		args = append(args, value)
	}
	return args, nil
}

func buildValuesSQL(rowCount, colCount int) string {
	rows := make([]string, rowCount)
	for i := 0; i < rowCount; i++ {
		ph := make([]string, colCount)
		base := i * colCount
		for j := 0; j < colCount; j++ {
			ph[j] = fmt.Sprintf("$%d", base+j+1)
		}
		rows[i] = "(" + strings.Join(ph, ", ") + ")"
	}
	return strings.Join(rows, ", ")
}

func columnSQL(fields []*fieldMeta) string {
	cols := make([]string, len(fields))
	for i, f := range fields {
		cols[i] = f.DBName
	}
	return strings.Join(cols, ", ")
}

// ---- 构建 scan 目标和后处理 ----

type scanContext struct {
	scanArgs    []any
	jsonBuffers [][]byte
	ptrSlots    []ptrScanSlot
}

func (t *Table[T]) prepareScan(val reflect.Value) *scanContext {
	sc := &scanContext{
		scanArgs:    make([]any, len(t.fields)),
		jsonBuffers: make([][]byte, len(t.fields)),
		ptrSlots:    make([]ptrScanSlot, 0, len(t.fields)),
	}

	for i, field := range t.fields {
		fieldVal := val.Field(field.Index)

		switch field.ScanKind {
		case scanJSON:
			sc.scanArgs[i] = &sc.jsonBuffers[i]

		case scanPtrString:
			var ns sql.NullString
			sc.scanArgs[i] = &ns
			sc.ptrSlots = append(sc.ptrSlots, ptrScanSlot{
				fieldIndex: field.Index,
				kind:       scanPtrString,
				elemType:   field.ElemType,
				target:     &ns,
			})

		case scanPtrBool:
			var nb sql.NullBool
			sc.scanArgs[i] = &nb
			sc.ptrSlots = append(sc.ptrSlots, ptrScanSlot{
				fieldIndex: field.Index,
				kind:       scanPtrBool,
				elemType:   field.ElemType,
				target:     &nb,
			})

		case scanPtrInt:
			var ni sql.NullInt64
			sc.scanArgs[i] = &ni
			sc.ptrSlots = append(sc.ptrSlots, ptrScanSlot{
				fieldIndex: field.Index,
				kind:       scanPtrInt,
				elemType:   field.ElemType,
				target:     &ni,
			})

		case scanPtrUint:
			var ni sql.NullInt64
			sc.scanArgs[i] = &ni
			sc.ptrSlots = append(sc.ptrSlots, ptrScanSlot{
				fieldIndex: field.Index,
				kind:       scanPtrUint,
				elemType:   field.ElemType,
				target:     &ni,
			})

		case scanPtrFloat:
			var nf sql.NullFloat64
			sc.scanArgs[i] = &nf
			sc.ptrSlots = append(sc.ptrSlots, ptrScanSlot{
				fieldIndex: field.Index,
				kind:       scanPtrFloat,
				elemType:   field.ElemType,
				target:     &nf,
			})

		case scanPtrTime:
			var nt sql.NullTime
			sc.scanArgs[i] = &nt
			sc.ptrSlots = append(sc.ptrSlots, ptrScanSlot{
				fieldIndex: field.Index,
				kind:       scanPtrTime,
				elemType:   field.ElemType,
				target:     &nt,
			})
		default:
			sc.scanArgs[i] = fieldVal.Addr().Interface()
		}
	}

	return sc
}

// 后置处理扫描赋值（JSON类 和 自定义逻辑类）
func (t *Table[T]) finalizeScan(val reflect.Value, sc *scanContext) error {
	// 处理指针类型
	for _, slot := range sc.ptrSlots {
		if err := slot.apply(val); err != nil {
			return err
		}
	}

	// 处理 JSON 类型
	for i, buf := range sc.jsonBuffers {
		if len(buf) == 0 {
			continue
		}

		field := t.fields[i]
		fieldVal := val.Field(field.Index)

		if fieldVal.Kind() == reflect.Ptr {
			if fieldVal.IsNil() {
				fieldVal.Set(reflect.New(fieldVal.Type().Elem()))
			}
			fieldVal = fieldVal.Elem()
		}

		if err := json.Unmarshal(buf, fieldVal.Addr().Interface()); err != nil {
			return fmt.Errorf("unmarshal json field %s failed: %w", field.Name, err)
		}
	}

	return nil
}

// ---- 列名拼接（复用） ----

func (t *Table[T]) allColumnSQL() string {
	cols := make([]string, len(t.fields))
	for i, f := range t.fields {
		cols[i] = f.DBName
	}
	return strings.Join(cols, ", ")
}

func (t *Table[T]) insertColumnSQL() string {
	cols := make([]string, len(t.insertFields))
	for i, f := range t.insertFields {
		cols[i] = f.DBName
	}
	return strings.Join(cols, ", ")
}

// ---- 多行扫描（复用逻辑） ----

// Rows 接口兼容 pgx 和 database/sql
type Rows interface {
	Next() bool
	Scan(dest ...any) error
	Err() error
}

func (t *Table[T]) scanRows(rows Rows) ([]*T, error) {
	var results []*T

	for rows.Next() {
		var entity T
		val := reflect.ValueOf(&entity).Elem()
		sc := t.prepareScan(val)

		if err := rows.Scan(sc.scanArgs...); err != nil {
			return nil, fmt.Errorf("scanRows scan error: %w", err)
		}

		if err := t.finalizeScan(val, sc); err != nil {
			return nil, err
		}

		results = append(results, &entity)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("scanRows iteration error: %w", err)
	}

	return results, nil
}

// ---- ptrScanSlot ----

type ptrScanSlot struct {
	fieldIndex int
	kind       fieldScanKind
	elemType   reflect.Type
	target     any
}

func (s ptrScanSlot) apply(val reflect.Value) error {
	fieldVal := val.Field(s.fieldIndex)

	switch s.kind {
	case scanPtrString:
		ns := s.target.(*sql.NullString)
		if !ns.Valid {
			fieldVal.Set(reflect.Zero(fieldVal.Type()))
			return nil
		}
		p := reflect.New(s.elemType)
		p.Elem().SetString(ns.String)
		fieldVal.Set(p)

	case scanPtrBool:
		nb := s.target.(*sql.NullBool)
		if !nb.Valid {
			fieldVal.Set(reflect.Zero(fieldVal.Type()))
			return nil
		}
		p := reflect.New(s.elemType)
		p.Elem().SetBool(nb.Bool)
		fieldVal.Set(p)

	case scanPtrInt:
		ni := s.target.(*sql.NullInt64)
		if !ni.Valid {
			fieldVal.Set(reflect.Zero(fieldVal.Type()))
			return nil
		}
		p := reflect.New(s.elemType)
		p.Elem().SetInt(ni.Int64)
		fieldVal.Set(p)

	case scanPtrUint:
		ni := s.target.(*sql.NullInt64)
		if !ni.Valid {
			fieldVal.Set(reflect.Zero(fieldVal.Type()))
			return nil
		}
		if ni.Int64 < 0 {
			return fmt.Errorf("cannot scan negative value %d into unsigned field", ni.Int64)
		}
		p := reflect.New(s.elemType)
		p.Elem().SetUint(uint64(ni.Int64))
		fieldVal.Set(p)

	case scanPtrFloat:
		nf := s.target.(*sql.NullFloat64)
		if !nf.Valid {
			fieldVal.Set(reflect.Zero(fieldVal.Type()))
			return nil
		}
		p := reflect.New(s.elemType)
		p.Elem().SetFloat(nf.Float64)
		fieldVal.Set(p)

	case scanPtrTime:
		nt := s.target.(*sql.NullTime)
		if !nt.Valid {
			fieldVal.Set(reflect.Zero(fieldVal.Type()))
			return nil
		}
		if s.elemType != nil {
			p := reflect.New(s.elemType)
			p.Elem().Set(reflect.ValueOf(nt.Time))
			fieldVal.Set(p)
		} else {
			fieldVal.Set(reflect.ValueOf(nt.Time))
		}
	}

	return nil
}

// ---- classify / getValue ----

func classifyScanKind(fieldType reflect.Type) (fieldScanKind, reflect.Type) {
	// 指针类型
	if fieldType.Kind() == reflect.Ptr {
		elem := fieldType.Elem()
		if elem.Kind() == reflect.Struct {
			if elem == timeType { // 时间类型
				return scanPtrTime, elem
			}
			// 先查 *T（指针接收者），再查 T（值接收者）
			if fieldType.Implements(scannerType) || fieldType.Implements(valuerType) {
				return scanDirect, nil
			}
			if elem.Implements(scannerType) || elem.Implements(valuerType) {
				return scanDirect, nil
			}
			return scanJSON, nil
		}

		// *slice, *map → JSON
		if elem.Kind() == reflect.Slice || elem.Kind() == reflect.Map {
			return scanJSON, nil
		}

		// *基础类型
		switch elem.Kind() {
		case reflect.String:
			return scanPtrString, elem
		case reflect.Bool:
			return scanPtrBool, elem
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			return scanPtrInt, elem
		case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
			return scanPtrUint, elem
		case reflect.Float32, reflect.Float64:
			return scanPtrFloat, elem
		default:
			return scanDirect, nil
		}
	}

	// 非指针 struct
	if fieldType.Kind() == reflect.Struct {
		if fieldType == timeType {
			return scanPtrTime, nil
		}
		// 值接收者
		if fieldType.Implements(scannerType) || fieldType.Implements(valuerType) {
			return scanDirect, nil
		}
		// 指针接收者
		if reflect.PointerTo(fieldType).Implements(scannerType) || reflect.PointerTo(fieldType).Implements(valuerType) {
			return scanDirect, nil
		}
		return scanJSON, nil
	}

	// Slice / Map → JSON（[]byte 除外）
	if fieldType.Kind() == reflect.Slice {
		if fieldType == reflect.TypeOf([]byte(nil)) {
			return scanDirect, nil
		}
		return scanJSON, nil
	}
	if fieldType.Kind() == reflect.Map {
		return scanJSON, nil
	}

	// 普通基础类型
	return scanDirect, nil
}

func getValue(fieldValue reflect.Value) (any, error) {
	// 指针解引用，解引用前先检查 Valuer
	for fieldValue.Kind() == reflect.Ptr {
		if fieldValue.IsNil() {
			return nil, nil
		}
		if fieldValue.Type().Implements(valuerType) {
			return fieldValue.Interface().(driver.Valuer).Value()
		}
		fieldValue = fieldValue.Elem()
	}

	// 非指针字段，检查指针接收者的 Valuer（如 func (s *MyType) Value()）
	if fieldValue.CanAddr() {
		if valuer, ok := fieldValue.Addr().Interface().(driver.Valuer); ok {
			return valuer.Value()
		}
	}

	// 值接收者的 Valuer
	if valuer, ok := fieldValue.Interface().(driver.Valuer); ok {
		return valuer.Value()
	}

	// time.Time 特殊处理
	if fieldValue.Type() == timeType {
		t := fieldValue.Interface().(time.Time)
		if t.IsZero() {
			return nil, nil
		}
		return t, nil
	}

	// struct → JSON
	if fieldValue.Kind() == reflect.Struct {
		jsonData, err := json.Marshal(fieldValue.Interface())
		if err != nil {
			return nil, fmt.Errorf("json marshal error: %w", err)
		}
		return jsonData, nil
	}

	// slice / map → JSON
	if fieldValue.Kind() == reflect.Slice || fieldValue.Kind() == reflect.Map {
		if fieldValue.IsNil() {
			return nil, nil
		}
		// []byte 直接传
		if fieldValue.Type() == reflect.TypeOf([]byte(nil)) {
			return fieldValue.Interface(), nil
		}
		jsonData, err := json.Marshal(fieldValue.Interface())
		if err != nil {
			return nil, fmt.Errorf("json marshal error: %w", err)
		}
		return jsonData, nil
	}

	return fieldValue.Interface(), nil
}
