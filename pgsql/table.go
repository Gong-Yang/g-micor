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

// ---- 查询条件构建 ----

type WhereBuilder struct {
	conditions []string
	args       []any
	orderBy    string
	limit      int
	offset     int
}

func Where(condition string, args ...any) *WhereBuilder {
	wb := &WhereBuilder{}
	return wb.And(condition, args...)
}

func (wb *WhereBuilder) And(condition string, args ...any) *WhereBuilder {
	wb.conditions = append(wb.conditions, condition)
	wb.args = append(wb.args, args...)
	return wb
}

func (wb *WhereBuilder) OrderBy(orderBy string) *WhereBuilder {
	wb.orderBy = orderBy
	return wb
}

func (wb *WhereBuilder) Limit(limit int) *WhereBuilder {
	wb.limit = limit
	return wb
}

func (wb *WhereBuilder) Offset(offset int) *WhereBuilder {
	wb.offset = offset
	return wb
}

// reindex 将条件中的 $1, $2... 占位符按照 startIdx 重新编号
// 这样在 InsertMany 等场景拼 SQL 时不会冲突
func (wb *WhereBuilder) buildSQL(startIdx int) (string, []any) {
	if wb == nil || len(wb.conditions) == 0 {
		return "", nil
	}

	// 重新编号占位符
	var reindexed []string
	argIdx := startIdx
	for _, cond := range wb.conditions {
		var newCond strings.Builder
		i := 0
		for i < len(cond) {
			if cond[i] == '$' {
				// 跳过原始的 $N
				j := i + 1
				for j < len(cond) && cond[j] >= '0' && cond[j] <= '9' {
					j++
				}
				newCond.WriteString(fmt.Sprintf("$%d", argIdx))
				argIdx++
				i = j
			} else {
				newCond.WriteByte(cond[i])
				i++
			}
		}
		reindexed = append(reindexed, newCond.String())
	}

	clause := " WHERE " + strings.Join(reindexed, " AND ")

	if wb.orderBy != "" {
		clause += " ORDER BY " + wb.orderBy
	}
	if wb.limit > 0 {
		clause += fmt.Sprintf(" LIMIT %d", wb.limit)
	}
	if wb.offset > 0 {
		clause += fmt.Sprintf(" OFFSET %d", wb.offset)
	}

	return clause, wb.args
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

func GetTable[T DBEntity]() *Table[T] {
	var t T
	tType := reflect.TypeOf(t)
	if tType == nil || tType.Kind() != reflect.Struct {
		panic("T must be a struct")
	}

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

// 将对象提取为insert的参数
func (t *Table[T]) extractInsertArgs(ctx context.Context, entity *T) ([]any, error) {
	val := reflect.ValueOf(entity).Elem()
	args := make([]any, 0, len(t.insertFields))

	for _, field := range t.insertFields {
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

// ---- InsertOne ----

func (t *Table[T]) InsertOne(ctx context.Context, entity *T) error {
	pool, err := PoolManager.Get(ctx)
	if err != nil {
		return err
	}

	args, err := t.extractInsertArgs(ctx, entity)
	if err != nil {
		return err
	}

	row := pool.QueryRow(ctx, t.insertOneSQL, args...)
	var returnedID int64
	if err := row.Scan(&returnedID); err != nil {
		slog.ErrorContext(ctx, "InsertOne error", "err", err)
		return err
	}

	reflect.ValueOf(entity).Elem().Field(t.pkField.Index).SetInt(returnedID)
	return nil
}

// ---- InsertMany ----

func (t *Table[T]) InsertMany(ctx context.Context, entities []*T) error {
	if len(entities) == 0 {
		return nil
	}

	pool, err := PoolManager.Get(ctx)
	if err != nil {
		return err
	}

	colCount := len(t.insertFields)
	allArgs := make([]any, 0, colCount*len(entities))
	valueGroups := make([]string, 0, len(entities))

	for rowIdx, entity := range entities {
		args, err := t.extractInsertArgs(ctx, entity)
		if err != nil {
			return err
		}
		allArgs = append(allArgs, args...)

		// 构建 ($1, $2, $3), ($4, $5, $6), ...
		placeholders := make([]string, colCount)
		base := rowIdx * colCount
		for j := 0; j < colCount; j++ {
			placeholders[j] = fmt.Sprintf("$%d", base+j+1)
		}
		valueGroups = append(valueGroups, "("+strings.Join(placeholders, ", ")+")")
	}

	query := fmt.Sprintf(
		"INSERT INTO %s (%s) VALUES %s RETURNING id",
		t.name,
		t.insertColumnSQL(),
		strings.Join(valueGroups, ", "),
	)

	rows, err := pool.Query(ctx, query, allArgs...)
	if err != nil {
		slog.ErrorContext(ctx, "InsertMany error", "err", err)
		return err
	}
	defer rows.Close()

	i := 0
	for rows.Next() {
		var returnedID int64
		if err := rows.Scan(&returnedID); err != nil {
			slog.ErrorContext(ctx, "InsertMany scan error", "err", err)
			return err
		}
		reflect.ValueOf(entities[i]).Elem().Field(t.pkField.Index).SetInt(returnedID)
		i++
	}

	return rows.Err()
}

// ---- FindByID ----

func (t *Table[T]) FindByID(ctx context.Context, id int64) (*T, error) {
	pool, err := PoolManager.Get(ctx)
	if err != nil {
		return nil, err
	}

	row := pool.QueryRow(ctx, t.selectOneSQL, id)

	var entity T
	val := reflect.ValueOf(&entity).Elem()
	sc := t.prepareScan(val)

	if err = row.Scan(sc.scanArgs...); err != nil {
		slog.ErrorContext(ctx, "FindByID error", "err", err)
		return nil, err
	}

	if err = t.finalizeScan(val, sc); err != nil {
		return nil, err
	}

	return &entity, nil
}

// ---- Find ----

func (t *Table[T]) Find(ctx context.Context, wb *WhereBuilder) ([]*T, error) {
	pool, err := PoolManager.Get(ctx)
	if err != nil {
		return nil, err
	}

	query := fmt.Sprintf("SELECT %s FROM %s", t.allColumnSQL(), t.name)

	whereClause, whereArgs := wb.buildSQL(1)
	query += whereClause

	rows, err := pool.Query(ctx, query, whereArgs...)
	if err != nil {
		slog.ErrorContext(ctx, "Find error", "err", err)
		return nil, err
	}
	defer rows.Close()

	return t.scanRows(rows)
}

// ---- FindPage ----

func (t *Table[T]) FindPage(ctx context.Context, wb *WhereBuilder, page, pageSize int) (*Page[T], error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 10
	}

	pool, err := PoolManager.Get(ctx)
	if err != nil {
		return nil, err
	}

	// 1. COUNT 查询
	countSQL := fmt.Sprintf("SELECT COUNT(*) FROM %s", t.name)
	whereClause, whereArgs := wb.buildSQL(1)
	countSQL += whereClause

	var total int64
	if err := pool.QueryRow(ctx, countSQL, whereArgs...).Scan(&total); err != nil {
		slog.ErrorContext(ctx, "FindPage count error", "err", err)
		return nil, err
	}

	result := &Page[T]{Total: total}

	if total == 0 {
		result.Items = []*T{}
		return result, nil
	}

	// 2. 数据查询：在 WHERE 子句基础上追加分页
	// 先用不带 LIMIT/OFFSET 的 whereBuilder 构建条件
	dataSQL := fmt.Sprintf("SELECT %s FROM %s", t.allColumnSQL(), t.name)

	// 复制 wb 并追加排序+分页
	dataWb := &WhereBuilder{
		conditions: wb.conditions,
		args:       wb.args,
		orderBy:    wb.orderBy,
		limit:      pageSize,
		offset:     (page - 1) * pageSize,
	}

	// 如果没有指定排序，默认按 id 排序保证分页稳定
	if dataWb.orderBy == "" {
		dataWb.orderBy = "id ASC"
	}

	dataClause, dataArgs := dataWb.buildSQL(1)
	dataSQL += dataClause

	rows, err := pool.Query(ctx, dataSQL, dataArgs...)
	if err != nil {
		slog.ErrorContext(ctx, "FindPage query error", "err", err)
		return nil, err
	}
	defer rows.Close()

	items, err := t.scanRows(rows)
	if err != nil {
		return nil, err
	}

	result.Items = items
	return result, nil
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
				return scanDirect, nil
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
