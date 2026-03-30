package pgsql

import (
	"fmt"
	"strings"
)

// ---- UpdateBuilder ----

type UpdateBuilder struct {
	sets []setEntry
}
type setEntry struct {
	column string
	expr   string // 非空时使用表达式而非 $N 占位符
	args   []any  // 该条目消耗的参数
}

func Set(column string, value any) *UpdateBuilder {
	ub := &UpdateBuilder{}
	return ub.Set(column, value)
}

func (ub *UpdateBuilder) Set(column string, value any) *UpdateBuilder {
	ub.sets = append(ub.sets, setEntry{
		column: column,
		args:   []any{value},
	})
	return ub
}

// SetExpr 支持自定义 SQL 表达式，用 $? 作为占位符标记
// 示例: ub.SetExpr("tags", "tags || $?::jsonb", serializedJSON)
func (ub *UpdateBuilder) SetExpr(column, expr string, args ...any) *UpdateBuilder {
	ub.sets = append(ub.sets, setEntry{
		column: column,
		expr:   expr,
		args:   args,
	})
	return ub
}

func (ub *UpdateBuilder) buildSQL(startIdx int) (string, []any, int) {
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
			// 普通 SET col = $N
			parts = append(parts, fmt.Sprintf("%s = $%d", entry.column, idx))
			allArgs = append(allArgs, entry.args[0])
			idx++
		} else {
			// 表达式模式：将 $? 替换为实际编号
			resolved := entry.expr
			for _, arg := range entry.args {
				resolved = strings.Replace(resolved, "$?", fmt.Sprintf("$%d", idx), 1)
				allArgs = append(allArgs, arg)
				idx++
			}
			parts = append(parts, fmt.Sprintf("%s = %s", entry.column, resolved))
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
