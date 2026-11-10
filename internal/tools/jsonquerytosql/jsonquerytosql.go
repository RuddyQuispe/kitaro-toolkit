// Package jsonquerytosql converts a "query JSON" document (select/from/
// where/groupBy/orderBy, with an optional recursively-nested unionAll, an
// optional root-level cte and join objects inside from) into a raw SQL
// statement. Each array element is treated as pre-built SQL — this
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
	CTE      *cteJSON   `json:"cte"`
	Select   []string   `json:"select"`
	From     []fromItem `json:"from"`
	Where    []string   `json:"where"`
	GroupBy  []string   `json:"groupBy"`
	OrderBy  stringList `json:"orderBy"`
	UnionAll *queryJSON `json:"unionAll"`
}

// stringList accepts either a JSON array of strings or a single string
// (the legacy format, e.g. "orderBy": "u.USER_LOGIN ASC"), which becomes a
// one-element list.
type stringList []string

func (l *stringList) UnmarshalJSON(data []byte) error {
	var single string
	if err := json.Unmarshal(data, &single); err == nil {
		*l = stringList{single}
		return nil
	}
	var list []string
	if err := json.Unmarshal(data, &list); err != nil {
		return fmt.Errorf("must be a string or an array of strings")
	}
	*l = list
	return nil
}

// cteJSON is a single common table expression, rendered as
// "WITH <name> AS (<definition>)" before the root query.
type cteJSON struct {
	Name       string     `json:"name"`
	Definition *queryJSON `json:"definition"`
}

// joinKeywords maps each accepted join key of a from object to its SQL
// keyword.
var joinKeywords = map[string]string{
	"innerJoin": "INNER JOIN",
	"leftJoin":  "LEFT JOIN",
	"rightJoin": "RIGHT JOIN",
	"fullJoin":  "FULL JOIN",
	"crossJoin": "CROSS JOIN",
	"join":      "JOIN",
}

// fromItem is either a plain table (a JSON string, e.g. "COM_CLIENT CLIENT")
// or a join object such as {"innerJoin": "T", "name": "alias", "on": [...]}.
// JoinKeyword is empty for plain tables.
type fromItem struct {
	Table       string
	Alias       string
	JoinKeyword string
	On          []string
}

func (f *fromItem) UnmarshalJSON(data []byte) error {
	// 1. Plain table: a JSON string.
	if err := json.Unmarshal(data, &f.Table); err == nil {
		return nil
	}

	// 2. Join object: exactly one join key, plus optional name and on.
	var obj map[string]json.RawMessage
	if err := json.Unmarshal(data, &obj); err != nil {
		return fmt.Errorf(`"from" items must be strings or join objects`)
	}
	for key, keyword := range joinKeywords {
		raw, ok := obj[key]
		if !ok {
			continue
		}
		if f.JoinKeyword != "" {
			return fmt.Errorf(`join object must have exactly one join key (innerJoin, leftJoin, rightJoin, fullJoin, crossJoin, join)`)
		}
		if err := json.Unmarshal(raw, &f.Table); err != nil {
			return fmt.Errorf("%q must be a string: %w", key, err)
		}
		f.JoinKeyword = keyword
	}
	if f.JoinKeyword == "" {
		return fmt.Errorf(`join object must have exactly one join key (innerJoin, leftJoin, rightJoin, fullJoin, crossJoin, join)`)
	}
	if raw, ok := obj["name"]; ok {
		if err := json.Unmarshal(raw, &f.Alias); err != nil {
			return fmt.Errorf(`join "name" must be a string: %w`, err)
		}
	}
	if raw, ok := obj["on"]; ok {
		if err := json.Unmarshal(raw, &f.On); err != nil {
			return fmt.Errorf(`join "on" must be an array of strings: %w`, err)
		}
	}
	if f.JoinKeyword == "CROSS JOIN" && len(f.On) > 0 {
		return fmt.Errorf(`"crossJoin" does not take "on" conditions`)
	}
	return nil
}

// Run parses a query JSON document and returns the equivalent raw SQL
// statement.
func Run(input string, opts map[string]string) (string, error) {
	var q queryJSON
	if err := json.Unmarshal([]byte(input), &q); err != nil {
		return "", fmt.Errorf("invalid JSON: %w", err)
	}

	var sb strings.Builder
	if q.CTE != nil {
		if err := writeCTE(&sb, q.CTE); err != nil {
			return "", err
		}
	}
	if err := writeQuery(&sb, &q, true); err != nil {
		return "", err
	}
	return sb.String(), nil
}

// writeCTE renders "WITH <name> AS (<definition>) ". The definition is a
// standalone query, so it's rendered as a root (its own orderBy is honored).
func writeCTE(sb *strings.Builder, cte *cteJSON) error {
	if cte.Name == "" {
		return fmt.Errorf(`"cte" must include a non-empty "name"`)
	}
	if cte.Definition == nil {
		return fmt.Errorf(`"cte" must include a "definition" query`)
	}
	sb.WriteString("WITH ")
	sb.WriteString(cte.Name)
	sb.WriteString(" AS (")
	if err := writeQuery(sb, cte.Definition, true); err != nil {
		return fmt.Errorf("cte %q: %w", cte.Name, err)
	}
	sb.WriteString(") ")
	return nil
}

// writeFrom renders the from list: plain tables are joined with ", " and
// join objects are appended as "<KEYWORD> <table> [alias] [ON a AND b]".
// ON conditions are joined as-is (no per-condition parens, unlike WHERE),
// so a condition containing OR must carry its own parentheses.
func writeFrom(sb *strings.Builder, items []fromItem) error {
	if items[0].JoinKeyword != "" {
		return fmt.Errorf(`the first "from" item must be a table, not a join`)
	}
	sb.WriteString(items[0].Table)
	for _, item := range items[1:] {
		if item.JoinKeyword == "" {
			sb.WriteString(", ")
			sb.WriteString(item.Table)
			continue
		}
		sb.WriteString(" ")
		sb.WriteString(item.JoinKeyword)
		sb.WriteString(" ")
		sb.WriteString(item.Table)
		if item.Alias != "" {
			sb.WriteString(" ")
			sb.WriteString(item.Alias)
		}
		if len(item.On) > 0 {
			sb.WriteString(" ON ")
			sb.WriteString(strings.Join(item.On, " AND "))
		}
	}
	return nil
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
	if err := writeFrom(sb, q.From); err != nil {
		return err
	}

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
