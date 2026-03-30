package pgsql

import (
	"fmt"
	"reflect"
	"strings"
)

// ---- 新增：参数标准化辅助函数 ----
// 作用：复用 getValue 的逻辑，把 Go 类型转成数据库驱动认识的类型
func toDBValue(arg any) (any, error) {
	if arg == nil {
		return nil, nil
	}
	// 调用你框架里已有的 getValue
	return getValue(reflect.ValueOf(arg))
}

// ---- UpdateBuilder 结构体 (保持不变) ----
type UpdateBuilder struct {
	sets []setEntry
}

type setEntry struct {
	column string
	expr   string
	args   []any
}

func Set(column string, value any) *UpdateBuilder {
	ub := &UpdateBuilder{}
	return ub.Set(column, value)
}
func SetExpr(expr string, args ...any) *UpdateBuilder {
	ub := &UpdateBuilder{}
	return ub.SetExpr(expr, args...)
}

// Set key value
func (ub *UpdateBuilder) Set(column string, value any) *UpdateBuilder {
	val, err := toDBValue(value)
	if err != nil {
		panic(fmt.Sprintf("UpdateBuilder.Set error: %v", err))
	}

	ub.sets = append(ub.sets, setEntry{
		column: column,
		args:   []any{val},
	})
	return ub
}

// SetExpr key expression
func (ub *UpdateBuilder) SetExpr(expr string, args ...any) *UpdateBuilder {
	processedArgs := make([]any, len(args))
	for i, arg := range args {
		val, err := toDBValue(arg)
		if err != nil {
			panic(fmt.Sprintf("UpdateBuilder.SetExpr error: %v", err))
		}
		processedArgs[i] = val
	}

	ub.sets = append(ub.sets, setEntry{
		expr: expr,
		args: processedArgs,
	})
	return ub
}

// buildSQL 完全不需要改！
func (ub *UpdateBuilder) buildSQL(startIdx int) (string, []any, int) {
	// ... (你的原逻辑，保持不变) ...
	// 它现在会拿到已经被 toDBValue 处理过的参数，直接用就行了
	if ub == nil || len(ub.sets) == 0 {
		return "", nil, startIdx
	}

	var (
		parts   []string
		allArgs []any
		idx     = startIdx
	)

	for _, entry := range ub.sets {
		if entry.expr == "" {
			parts = append(parts, fmt.Sprintf("%s = $%d", entry.column, idx))
			allArgs = append(allArgs, entry.args[0])
			idx++
		} else {
			resolved := entry.expr
			for _, arg := range entry.args {
				resolved = strings.Replace(resolved, "$1", fmt.Sprintf("$%d", idx), 1)
				allArgs = append(allArgs, arg)
				idx++
			}
			parts = append(parts, resolved)
		}
	}

	clause := "SET " + strings.Join(parts, ", ")
	return clause, allArgs, idx
}

//// 1. 根据 ID 更新整个实体
//user := &User{ID: 1, Name: "new_name", Age: 30}
//err := userTable.UpdateByID(ctx, user)
//
//// 2. 条件更新指定字段
//affected, err := userTable.Update(ctx,
//Set("name", "new_name").Set("age", 30),
//Where("status = $1", 1).And("created_at < $1", someTime),
//)
