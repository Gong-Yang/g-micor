package main

import (
	"context"
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"log/slog"
	"reflect"
	"strings"
	"time"

	"github.com/Gong-Yang/g-micor/syncx"
)

type Table[T DBEntity] struct {
	Name         string
	PKField      *fieldMeta
	Fields       []*fieldMeta
	InsertOneSQL string
}
type DBEntity interface {
	TableName() string
}

type fieldMeta struct {
	Name         string
	DBName       string
	Index        int
	IsPrimaryKey bool // 新增：标记是否为主键
}

var tableStore = syncx.NewResourceManager[any]()

func GetTable[T DBEntity]() *Table[T] {
	var t T
	tType := reflect.TypeOf(t)

	// 处理 T 是指针的情况，虽然泛型约束建议 T 是结构体，但做个保护更安全
	if tType.Kind() != reflect.Struct {
		panic("T must be a struct")
	}
	findId := false
	tableObj, _ := tableStore.GetResource(tType.String(), func() (any, error) {
		var fields []*fieldMeta
		var columns []string
		var placeholders []string
		var pkField *fieldMeta
		for i := range tType.NumField() {
			field := tType.Field(i)

			// 过滤掉未导出字段
			if !field.IsExported() {
				continue
			}

			dbName := field.Tag.Get("db")
			if dbName == "-" { // 支持 `-` 忽略字段
				continue
			}
			if dbName == "" {
				dbName = field.Name
			}

			// 主键约定叫id 并且是GENERATED ALWAYS AS IDENTITY
			isPK := strings.ToLower(dbName) == "id"
			filedInfo := &fieldMeta{
				Name:         field.Name,
				DBName:       dbName,
				Index:        i,
				IsPrimaryKey: isPK,
			}

			if isPK {
				findId = true
				if field.Type.Kind() != reflect.Int64 {
					return nil, fmt.Errorf("db id must be int64")
				}
				pkField = filedInfo
			} else {
				columns = append(columns, dbName)
				// 占位符索引从1开始
				placeholders = append(placeholders, fmt.Sprintf("$%d", len(columns)))
			}
			fields = append(fields, filedInfo)

		}
		if !findId {
			return nil, fmt.Errorf("table %s must have a id field", tType.Name())
		}

		tableName := t.TableName()
		return &Table[T]{
			Name:    tableName,
			Fields:  fields,
			PKField: pkField,
			InsertOneSQL: fmt.Sprintf(
				"INSERT INTO %s (%s) VALUES (%s) RETURNING id",
				tableName,
				strings.Join(columns, ", "),
				strings.Join(placeholders, ", "),
			),
		}, nil
	})
	return tableObj.(*Table[T])
}

func (t *Table[T]) InsertOne(ctx context.Context, entity *T) error {
	// 获取数据库连接
	pool, err := PoolManager.Get(ctx)
	if err != nil {
		return err
	}

	// 获取参数
	val := reflect.ValueOf(entity).Elem()
	var args []any
	for _, field := range t.Fields {
		if field.IsPrimaryKey { // 跳过主键字段
			continue
		}
		fieldValue := val.Field(field.Index)
		value, err := getValue(fieldValue)
		if err != nil {
			slog.ErrorContext(ctx, "get value error", "field", field.Name, "err", err)
			return err
		}
		args = append(args, value)
	}

	// 执行 SQL
	row := pool.QueryRow(ctx, t.InsertOneSQL, args...) //将ID写回entity
	var returnedID int64                               // 返回的ID
	if err := row.Scan(&returnedID); err != nil {
		return err
	}
	val.Field(t.PKField.Index).SetInt(returnedID)
	return nil
}

var timeType = reflect.TypeOf(time.Time{})

// getValue 获取字段值，处理指针、时间、JSONB 等类型
func getValue(fieldValue reflect.Value) (goValue any, err error) {
	// 循环解引用指针，处理 nil 情况
	for {
		if fieldValue.Kind() == reflect.Ptr {
			if fieldValue.IsNil() {
				return nil, nil
			}
			fieldValue = fieldValue.Elem()
			continue
		}
		break
	}

	// 特殊处理 time.Time，防止被当做普通 Struct 处理
	if fieldValue.Type() == timeType {
		return fieldValue.Interface(), nil
	}

	// 处理实现 driver.Valuer 接口的类型（比如自定义的 Enum、JSON 类型）
	// pgx 原生支持很多类型，但如果你自定义了 Value() 方法，优先调用
	if valuer, ok := fieldValue.Interface().(driver.Valuer); ok {
		return valuer.Value()
	}

	// 处理 Struct (通常是 JSONB 或嵌套结构体)
	if fieldValue.Kind() == reflect.Struct {
		jsonData, err := json.Marshal(fieldValue.Interface())
		if err != nil {
			return nil, fmt.Errorf("json marshal error: %w", err)
		}
		return jsonData, nil
	}

	// 默认情况：直接返回接口值
	return fieldValue.Interface(), nil
}
