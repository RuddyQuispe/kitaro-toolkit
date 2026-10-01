// Package jsonquerytosql converts a "query JSON" document (select/from/
// where/groupBy/orderBy, with an optional recursively-nested unionAll) into
// a raw SQL statement. Each array element is treated as pre-built SQL — this
// package only assembles keywords and separators, it never parses SQL. The
// frontend pretty-prints the result via sql-formatter.
package jsonquerytosql

import (
	"encoding/json"
	"fmt"
	"strings"

	"kitaro-toolkit/internal/tools"
)

func init() {
	tools.Register(&tools.Tool{ID: "jsonquerytosql", Name: "JSON Query to SQL", Run: Run})
}

type queryJSON struct {
	Select   []string   `json:"select"`
	From     []string   `json:"from"`
	Where    []string   `json:"where"`
	GroupBy  []string   `json:"groupBy"`
	OrderBy  []string   `json:"orderBy"`
	UnionAll *queryJSON `json:"unionAll"`
}

// Run parses a query JSON document and returns the equivalent raw SQL
// statement.
func Run(input string, opts map[string]string) (string, error) {
	var q queryJSON
	if err := json.Unmarshal([]byte(input), &q); err != nil {
		return "", fmt.Errorf("invalid JSON: %w", err)
	}

	var sb strings.Builder
	if err := writeQuery(&sb, &q, true); err != nil {
		return "", err
	}
	return sb.String(), nil
}

// writeQuery renders q into sb. orderBy is only honored at the root level —
// SQL disallows ORDER BY on intermediate branches of a UNION ALL chain, so
// it's written once, after the whole recursive chain has been assembled.
func writeQuery(sb *strings.Builder, q *queryJSON, isRoot bool) error {
	if len(q.Select) == 0 {
		return fmt.Errorf(`query JSON must include a non-empty "select" array`)
	}
	if len(q.From) == 0 {
		return fmt.Errorf(`query JSON must include a non-empty "from" array`)
	}

	sb.WriteString("SELECT ")
	sb.WriteString(strings.Join(q.Select, ", "))
	sb.WriteString(" FROM ")
	sb.WriteString(strings.Join(q.From, ", "))

	if len(q.Where) > 0 {
		sb.WriteString(" WHERE ")
		// Each condition is wrapped in its own parens — joining with a bare
		// " AND " would silently change meaning for a condition containing
		// an OR, due to operator precedence. Matches the Java query-JSON
		// builder's "where (a) and (b)" convention.
		wrapped := make([]string, len(q.Where))
		for i, w := range q.Where {
			wrapped[i] = "(" + w + ")"
		}
		sb.WriteString(strings.Join(wrapped, " AND "))
	}

	if len(q.GroupBy) > 0 {
		sb.WriteString(" GROUP BY ")
		sb.WriteString(strings.Join(q.GroupBy, ", "))
	}

	if q.UnionAll != nil {
		sb.WriteString(" UNION ALL ")
		if err := writeQuery(sb, q.UnionAll, false); err != nil {
			return err
		}
	}

	if isRoot && len(q.OrderBy) > 0 {
		sb.WriteString(" ORDER BY ")
		sb.WriteString(strings.Join(q.OrderBy, ", "))
	}

	return nil
}
