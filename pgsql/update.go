package pgsql

import (
	"fmt"
	"strings"
)

// ---- UpdateBuilder ----

type UpdateBuilder struct {
	sets []string
	args []any
}

func Set(column string, value any) *UpdateBuilder {
	ub := &UpdateBuilder{}
	return ub.Set(column, value)
}

func (ub *UpdateBuilder) Set(column string, value any) *UpdateBuilder {
	ub.sets = append(ub.sets, column)
	ub.args = append(ub.args, value)
	return ub
}

// buildSQL 生成 SET 子句，占位符从 startIdx 开始编号
// 返回: "SET col1 = $1, col2 = $2", args, nextIdx
func (ub *UpdateBuilder) buildSQL(startIdx int) (string, []any, int) {
	if ub == nil || len(ub.sets) == 0 {
		return "", nil, startIdx
	}

	parts := make([]string, len(ub.sets))
	for i, col := range ub.sets {
		parts[i] = fmt.Sprintf("%s = $%d", col, startIdx+i)
	}

	nextIdx := startIdx + len(ub.sets)
	clause := "SET " + strings.Join(parts, ", ")
	return clause, ub.args, nextIdx
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
