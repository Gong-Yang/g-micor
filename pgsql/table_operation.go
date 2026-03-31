package pgsql

import (
	"context"
	"fmt"
	"log/slog"
	"reflect"
	"strings"
)

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

func (t *Table[T]) FindOne(ctx context.Context, wb *WhereBuilder) (*T, error) {
	pool, err := PoolManager.Get(ctx)
	if err != nil {
		return nil, err
	}

	query := fmt.Sprintf("SELECT %s FROM %s", t.allColumnSQL(), t.name)

	whereClause, whereArgs := wb.buildSQL(1)
	query = query + whereClause + " LIMIT 1"

	row := pool.QueryRow(ctx, query, whereArgs...)
	var entity T
	val := reflect.ValueOf(&entity).Elem()
	sc := t.prepareScan(val)
	if err = row.Scan(sc.scanArgs...); err != nil {
		//slog.InfoContext(ctx, "FindOne error", "err", err)
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
func (t *Table[T]) Count(ctx context.Context, wb *WhereBuilder) (int64, error) {
	pool, err := PoolManager.Get(ctx)
	if err != nil {
		return 0, err
	}
	newWb := &WhereBuilder{
		conditions: wb.conditions,
		args:       wb.args,
	}
	// 1. COUNT 查询
	countSQL := fmt.Sprintf("SELECT COUNT(*) FROM %s", t.name)
	whereClause, whereArgs := newWb.buildSQL(1)
	countSQL += whereClause

	var total int64
	if err := pool.QueryRow(ctx, countSQL, whereArgs...).Scan(&total); err != nil {
		slog.ErrorContext(ctx, "FindPage count error", "err", err)
		return 0, err
	}
	return total, nil
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
	count, err := t.Count(ctx, wb)
	if err != nil {
		slog.ErrorContext(ctx, "FindPage count error", "err", err)
		return nil, err
	}

	result := &Page[T]{Total: count}

	if count == 0 {
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

// ---- UpdateByID ----

func (t *Table[T]) UpdateByID(ctx context.Context, entity *T) error {
	pool, err := PoolManager.Get(ctx)
	if err != nil {
		return err
	}

	val := reflect.ValueOf(entity).Elem()
	id := val.Field(t.pkField.Index).Int()
	if id == 0 {
		return fmt.Errorf("UpdateByID: entity id must not be zero")
	}

	// 构建 SET 子句（排除 id）
	setClauses := make([]string, len(t.insertFields))
	args := make([]any, 0, len(t.insertFields)+1)

	for i, field := range t.insertFields {
		setClauses[i] = fmt.Sprintf("%s = $%d", field.DBName, i+1)
		fieldValue := val.Field(field.Index)
		value, err := getValue(fieldValue)
		if err != nil {
			slog.ErrorContext(ctx, "UpdateByID get value error", "field", field.Name, "err", err)
			return err
		}
		args = append(args, value)
	}

	// id 作为最后一个参数
	args = append(args, id)
	query := fmt.Sprintf(
		"UPDATE %s SET %s WHERE id = $%d",
		t.name,
		strings.Join(setClauses, ", "),
		len(t.insertFields)+1,
	)

	cmdTag, err := pool.Exec(ctx, query, args...)
	if err != nil {
		slog.ErrorContext(ctx, "UpdateByID error", "err", err)
		return err
	}

	if cmdTag.RowsAffected() == 0 {
		return fmt.Errorf("UpdateByID: no rows affected, id=%d", id)
	}

	return nil
}

// ---- Update ----

func (t *Table[T]) Update(ctx context.Context, ub *UpdateBuilder, wb *WhereBuilder) (int64, error) {
	if ub == nil || len(ub.sets) == 0 {
		return 0, fmt.Errorf("Update: nothing to set")
	}
	if wb == nil || len(wb.conditions) == 0 {
		return 0, fmt.Errorf("Update: where condition is required to prevent full table update")
	}

	pool, err := PoolManager.Get(ctx)
	if err != nil {
		return 0, err
	}

	// SET 子句从 $1 开始
	setClause, setArgs, nextIdx := ub.buildSQL(1)

	// WHERE 子句紧接着 SET 的编号
	whereClause, whereArgs := wb.buildSQL(nextIdx)

	query := fmt.Sprintf("UPDATE %s %s%s", t.name, setClause, whereClause)

	// 合并参数
	allArgs := make([]any, 0, len(setArgs)+len(whereArgs))
	allArgs = append(allArgs, setArgs...)
	allArgs = append(allArgs, whereArgs...)

	cmdTag, err := pool.Exec(ctx, query, allArgs...)
	if err != nil {
		slog.ErrorContext(ctx, "Update error", "err", err)
		return 0, err
	}

	return cmdTag.RowsAffected(), nil
}
