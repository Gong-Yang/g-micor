package pgsql

import (
	"fmt"
	"strings"
)

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

func OrderBy(condition string) *WhereBuilder {
	wb := &WhereBuilder{}
	return wb.OrderBy(condition)
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
	if wb == nil || (len(wb.conditions) == 0 && wb.orderBy == "" && wb.limit == 0) {
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

	clause := ""
	if len(reindexed) > 0 {
		clause = " WHERE " + strings.Join(reindexed, " AND ")
	}

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
