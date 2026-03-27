package main

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
	TableName() string
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
	insertFields []*fieldMeta
	insertOneSQL string
	selectOneSQL string
}

var (
	tableStore  = syncx.NewResourceManager[any]()
	timeType    = reflect.TypeOf(time.Time{})
	valuerType  = reflect.TypeOf((*driver.Valuer)(nil)).Elem()
	scannerType = reflect.TypeOf((*sql.Scanner)(nil)).Elem()
	// rawBytesType 已移除：sql.RawBytes 是 []byte（Kind=Slice），不会进 Struct 分支
)

func GetTable[T DBEntity]() *Table[T] {
	var t T
	tType := reflect.TypeOf(t)
	if tType == nil || tType.Kind() != reflect.Struct {
		panic("T must be a struct")
	}

	// 修复 #13：用完整包路径作为缓存 key，防止跨包同名 struct 冲突
	cacheKey := tType.PkgPath() + "." + tType.Name()

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

		// 修复 #6：只保留一个检查
		if pkField == nil {
			return nil, fmt.Errorf("table %s must have an id field", tType.Name())
		}

		tableName := t.TableName()
		return &Table[T]{
			name:         tableName,
			fields:       fields,
			insertFields: insertFields,
			pkField:      pkField,
			insertOneSQL: fmt.Sprintf(
				"INSERT INTO %s (%s) VALUES (%s) RETURNING id",
				tableName,
				strings.Join(columns, ", "),
				strings.Join(placeholders, ", "),
			),
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

func (t *Table[T]) InsertOne(ctx context.Context, entity *T) error {
	pool, err := PoolManager.Get(ctx)
	if err != nil {
		return err
	}

	val := reflect.ValueOf(entity).Elem()
	args := make([]any, 0, len(t.insertFields))

	for _, field := range t.insertFields {
		fieldValue := val.Field(field.Index)
		value, err := getValue(fieldValue)
		if err != nil {
			slog.ErrorContext(ctx, "get value error", "field", field.Name, "err", err)
			return err
		}
		args = append(args, value)
	}

	row := pool.QueryRow(ctx, t.insertOneSQL, args...)
	var returnedID int64
	if err := row.Scan(&returnedID); err != nil {
		slog.ErrorContext(ctx, "InsertOne error", "err", err)
		return err
	}

	val.Field(t.pkField.Index).SetInt(returnedID)
	return nil
}

func (t *Table[T]) FindByID(ctx context.Context, id int64) (res *T, err error) {
	pool, err := PoolManager.Get(ctx)
	if err != nil {
		return nil, err
	}

	row := pool.QueryRow(ctx, t.selectOneSQL, id)

	var entity T
	val := reflect.ValueOf(&entity).Elem()

	scanArgs := make([]any, len(t.fields))
	jsonBuffers := make([][]byte, len(t.fields))
	ptrSlots := make([]ptrScanSlot, 0, len(t.fields))

	for i, field := range t.fields {
		fieldVal := val.Field(field.Index)

		switch field.ScanKind {
		case scanJSON:
			scanArgs[i] = &jsonBuffers[i]

		case scanPtrString:
			var ns sql.NullString
			scanArgs[i] = &ns
			ptrSlots = append(ptrSlots, ptrScanSlot{
				fieldIndex: field.Index,
				kind:       scanPtrString,
				elemType:   field.ElemType,
				target:     &ns,
			})

		case scanPtrBool:
			var nb sql.NullBool
			scanArgs[i] = &nb
			ptrSlots = append(ptrSlots, ptrScanSlot{
				fieldIndex: field.Index,
				kind:       scanPtrBool,
				elemType:   field.ElemType,
				target:     &nb,
			})

		case scanPtrInt:
			var ni sql.NullInt64
			scanArgs[i] = &ni
			ptrSlots = append(ptrSlots, ptrScanSlot{
				fieldIndex: field.Index,
				kind:       scanPtrInt,
				elemType:   field.ElemType,
				target:     &ni,
			})

		case scanPtrUint:
			var ni sql.NullInt64
			scanArgs[i] = &ni
			ptrSlots = append(ptrSlots, ptrScanSlot{
				fieldIndex: field.Index,
				kind:       scanPtrUint,
				elemType:   field.ElemType,
				target:     &ni,
			})

		case scanPtrFloat:
			var nf sql.NullFloat64
			scanArgs[i] = &nf
			ptrSlots = append(ptrSlots, ptrScanSlot{
				fieldIndex: field.Index,
				kind:       scanPtrFloat,
				elemType:   field.ElemType,
				target:     &nf,
			})

		default:
			scanArgs[i] = fieldVal.Addr().Interface()
		}
	}

	if err = row.Scan(scanArgs...); err != nil {
		slog.ErrorContext(ctx, "FindByID error", "err", err)
		return nil, err
	}

	for _, slot := range ptrSlots {
		if err := slot.apply(val); err != nil {
			return nil, err
		}
	}

	for i, buf := range jsonBuffers {
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
			return nil, fmt.Errorf("unmarshal json field %s failed: %w", field.Name, err)
		}
	}

	return &entity, nil
}

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
	}

	return nil
}

func classifyScanKind(fieldType reflect.Type) (fieldScanKind, reflect.Type) {
	// *T
	if fieldType.Kind() == reflect.Ptr {
		elem := fieldType.Elem()

		// *struct
		if elem.Kind() == reflect.Struct {
			if elem == timeType {
				return scanDirect, nil
			}
			// 修复 #7：先查 *T（指针接收者），再查 T（值接收者）
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
			return scanDirect, nil
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

	// 修复 #8：Slice / Map → JSON（[]byte 除外）
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
		// 修复 #4：指针接收者的 Valuer 在解引用前捕获
		if fieldValue.Type().Implements(valuerType) {
			return fieldValue.Interface().(driver.Valuer).Value() // 修复：不要多加 nil
		}
		fieldValue = fieldValue.Elem()
	}

	// 修复：非指针字段，检查指针接收者的 Valuer（如 func (s *MyType) Value()）
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

	// 修复 #8：slice / map → JSON
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
