// Package sqltojsonquery converts a raw SQL statement into the "query JSON"
// document shape used by internal/tools/jsonquerytosql (select/from/where/
// groupBy/orderBy, with an optional recursively-nested unionAll, an optional
// leading WITH ... AS (...) cte and join objects inside from). It is the
// reverse of that package: instead of assembling opaque fragments into SQL,
// it splits SQL into fragments at the clause level. It never parses
// expressions — only enough structure (parentheses, string literals, clause
// keywords) to find safe split points.
package sqltojsonquery

import (
	"encoding/json"
	"fmt"
	"strings"

	"kitaro-toolkit/internal/tools"
)

func init() {
	tools.Register(&tools.Tool{ID: "sqltojsonquery", Name: "SQL to JSON Query", Run: Run})
}

type queryJSON struct {
	CTE      *cteJSON   `json:"cte,omitempty"`
	Select   []string   `json:"select,omitempty"`
	From     []fromItem `json:"from,omitempty"`
	Where    []string   `json:"where,omitempty"`
	GroupBy  []string   `json:"groupBy,omitempty"`
	OrderBy  []string   `json:"orderBy,omitempty"`
	UnionAll *queryJSON `json:"unionAll,omitempty"`
}

// cteJSON is a single "WITH <name> AS (<definition>)" common table
// expression.
type cteJSON struct {
	Name       string     `json:"name"`
	Definition *queryJSON `json:"definition"`
}

// fromItem is one from entry: a plain table (JoinKey empty, marshaled as a
// JSON string) or a join, marshaled as {"<joinKey>": table, "name": alias,
// "on": [...]} — the shape jsonquerytosql accepts.
type fromItem struct {
	JoinKey string
	Table   string
	Alias   string
	On      []string
}

// joinPatterns lists the top-level join keyword sequences and their JSON
// join key. Longer sequences come first so "LEFT JOIN" is never matched as
// a bare "JOIN" with a dangling "LEFT" on the previous table.
var joinPatterns = []struct {
	words []string
	key   string
}{
	{[]string{"INNER", "JOIN"}, "innerJoin"},
	{[]string{"LEFT", "OUTER", "JOIN"}, "leftJoin"},
	{[]string{"LEFT", "JOIN"}, "leftJoin"},
	{[]string{"RIGHT", "OUTER", "JOIN"}, "rightJoin"},
	{[]string{"RIGHT", "JOIN"}, "rightJoin"},
	{[]string{"FULL", "OUTER", "JOIN"}, "fullJoin"},
	{[]string{"FULL", "JOIN"}, "fullJoin"},
	{[]string{"JOIN"}, "join"},
}

// MarshalJSON writes plain tables as strings and joins with the join key
// first, then name and on (a map would sort the keys alphabetically).
func (f fromItem) MarshalJSON() ([]byte, error) {
	if f.JoinKey == "" {
		return json.Marshal(f.Table)
	}
	var sb strings.Builder
	key, _ := json.Marshal(f.JoinKey)
	table, _ := json.Marshal(f.Table)
	sb.WriteString("{" + string(key) + ":" + string(table))
	if f.Alias != "" {
		alias, _ := json.Marshal(f.Alias)
		sb.WriteString(`,"name":` + string(alias))
	}
	if len(f.On) > 0 {
		on, _ := json.Marshal(f.On)
		sb.WriteString(`,"on":` + string(on))
	}
	sb.WriteString("}")
	return []byte(sb.String()), nil
}

// UnmarshalJSON reads back what MarshalJSON writes.
func (f *fromItem) UnmarshalJSON(data []byte) error {
	if err := json.Unmarshal(data, &f.Table); err == nil {
		return nil
	}
	var obj struct {
		Name string   `json:"name"`
		On   []string `json:"on"`
	}
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	if err := json.Unmarshal(data, &obj); err != nil {
		return err
	}
	f.Alias, f.On = obj.Name, obj.On
	for _, p := range joinPatterns {
		if v, ok := raw[p.key]; ok {
			f.JoinKey = p.key
			return json.Unmarshal(v, &f.Table)
		}
	}
	return fmt.Errorf("from object without a join key")
}

// Run parses a raw SQL statement and returns the equivalent query JSON
// document, indented for readability.
func Run(input string, opts map[string]string) (string, error) {
	sql := strings.TrimSpace(input)
	if sql == "" {
		return "", fmt.Errorf("SQL input is empty")
	}
	if err := checkBalanced(sql); err != nil {
		return "", err
	}

	q, err := parseStatement(sql)
	if err != nil {
		return "", err
	}

	out, err := json.MarshalIndent(q, "", "  ")
	if err != nil {
		return "", err
	}
	return string(out), nil
}

// checkBalanced validates that parentheses and string literals in sql are
// well-formed before any clause splitting is attempted, so later scans can
// safely assume depth 0 at the start of any top-level substring.
func checkBalanced(sql string) error {
	depth := 0
	inString := false
	for i := 0; i < len(sql); i++ {
		c := sql[i]
		if inString {
			if c == '\'' {
				if i+1 < len(sql) && sql[i+1] == '\'' {
					i++
					continue
				}
				inString = false
			}
			continue
		}
		switch c {
		case '\'':
			inString = true
		case '(':
			depth++
		case ')':
			depth--
			if depth < 0 {
				return fmt.Errorf("unbalanced parentheses in SQL")
			}
		}
	}
	if inString {
		return fmt.Errorf("unterminated string literal in SQL")
	}
	if depth != 0 {
		return fmt.Errorf("unbalanced parentheses in SQL")
	}
	return nil
}

// parseStatement parses an optional leading "WITH <name> AS (<query>)"
// into q.CTE, then the main query. Only one CTE is supported, matching the
// forward tool's single "cte" object.
func parseStatement(sql string) (*queryJSON, error) {
	withEnd, hasWith := matchKeywordAt(sql, 0, []string{"WITH"})
	if !hasWith {
		return parseFull(sql)
	}

	// 1. Name, AS and the parenthesized definition.
	i := skipSpace(sql, withEnd)
	nameStart := i
	for i < len(sql) && isWordChar(sql[i]) {
		i++
	}
	name := sql[nameStart:i]
	asStart := skipSpace(sql, i)
	asEnd, hasAs := matchKeywordAt(sql, asStart, []string{"AS"})
	open := skipSpace(sql, asEnd)
	if name == "" || !hasAs || open >= len(sql) || sql[open] != '(' {
		return nil, fmt.Errorf("WITH must have the form: WITH <name> AS (<query>)")
	}
	closeIdx, err := matchingParen(sql, open)
	if err != nil {
		return nil, err
	}
	definition, err := parseFull(strings.TrimSpace(sql[open+1 : closeIdx]))
	if err != nil {
		return nil, fmt.Errorf("cte %q: %w", name, err)
	}

	// 2. The main query after the CTE.
	rest := strings.TrimSpace(sql[closeIdx+1:])
	if strings.HasPrefix(rest, ",") {
		return nil, fmt.Errorf("only one CTE is supported in WITH")
	}
	q, err := parseFull(rest)
	if err != nil {
		return nil, err
	}
	q.CTE = &cteJSON{Name: name, Definition: definition}
	return q, nil
}

// parseFull splits sql on top-level UNION ALL into branches, peels the
// root-only trailing ORDER BY off, and parses each branch into a queryJSON,
// chaining them via UnionAll.
//
// Two UNION ALL shapes are supported: the flat form our own forward tool
// produces ("A UNION ALL SELECT ... UNION ALL SELECT ..."), and the
// right-nested, parenthesized form the Java query-JSON builder in
// tesabiz-utils-r2 produces ("A union all (B union all (C))" — see
// splitUnionAll). In the parenthesized form, root ORDER BY (and an optional
// branch alias) sits after the outermost closing paren; splitUnionAll
// reports that as trailing. In the flat form, ORDER BY is simply appended
// after the last branch's own clauses, with nothing to report as trailing.
func parseFull(sql string) (*queryJSON, error) {
	branches, trailing, err := splitUnionAll(sql)
	if err != nil {
		return nil, err
	}

	orderByClause := trailing
	if orderByClause == "" {
		last := branches[len(branches)-1]
		if obStart, obEnd, found := findFirstTopLevel(last, 0, []string{"ORDER", "BY"}); found {
			orderByClause = last[obEnd:]
			branches[len(branches)-1] = last[:obStart]
		}
	} else if obEnd, ok := matchKeywordAt(orderByClause, 0, []string{"ORDER", "BY"}); ok {
		orderByClause = orderByClause[obEnd:]
	} else {
		return nil, fmt.Errorf("unexpected trailing text after UNION ALL: %q", orderByClause)
	}

	var root, prev *queryJSON
	for i, branch := range branches {
		q, err := parseBranch(branch)
		if err != nil {
			return nil, err
		}
		if i == 0 {
			root = q
		} else {
			prev.UnionAll = q
		}
		prev = q
	}

	if orderByClause != "" {
		root.OrderBy = splitTopLevelComma(orderByClause)
	}
	return root, nil
}

// splitUnionAll splits sql on every top-level "UNION ALL" occurrence,
// flattening the resulting branches left-to-right. A bare "UNION" (without
// ALL) is rejected, since the query JSON model only supports UNION ALL
// chains.
//
// Two shapes are recognized after "UNION ALL":
//   - flat (our own forward tool's output): the next branch follows
//     directly, e.g. "A UNION ALL SELECT ...".
//   - parenthesized (the Java query-JSON builder's output): the *entire*
//     rest of the chain is wrapped in one paren group, nested to the right,
//     e.g. "A union all (B union all (C))". The paren content is split
//     recursively (flattening further nested UNION ALLs), and an optional
//     branch alias (e.g. "U1") right after the closing paren is skipped.
//
// trailing is any text left over after the last branch/paren-group that
// isn't itself part of the union structure — in practice, the root's
// ORDER BY when the parenthesized shape was used (parseFull decides what to
// do with it).
func splitUnionAll(sql string) (branches []string, trailing string, err error) {
	pos := 0
	for {
		unionStart, unionEnd, found := findFirstTopLevel(sql, pos, []string{"UNION"})
		if !found {
			branches = append(branches, sql[pos:])
			return branches, "", nil
		}
		_, allEnd, ok := matchKeywordAtSkippingSpace(sql, unionEnd, "ALL")
		if !ok {
			return nil, "", fmt.Errorf("only UNION ALL is supported (found UNION without ALL)")
		}
		branches = append(branches, sql[pos:unionStart])

		i := allEnd
		for i < len(sql) && isSpace(sql[i]) {
			i++
		}
		if i < len(sql) && sql[i] == '(' {
			closeIdx, cerr := matchingParen(sql, i)
			if cerr != nil {
				return nil, "", cerr
			}
			innerBranches, innerTrailing, ierr := splitUnionAll(sql[i+1 : closeIdx])
			if ierr != nil {
				return nil, "", ierr
			}
			if innerTrailing != "" {
				return nil, "", fmt.Errorf("ORDER BY inside a non-root UNION ALL branch is not supported: %q", innerTrailing)
			}
			branches = append(branches, innerBranches...)
			pos = skipOptionalAlias(sql, closeIdx+1)
			return branches, strings.TrimSpace(sql[pos:]), nil
		}
		pos = allEnd
	}
}

// matchingParen returns the index of the ')' matching the '(' at openIdx,
// tracking string literals so a quoted paren doesn't confuse depth
// tracking. Input is assumed pre-validated by checkBalanced.
func matchingParen(s string, openIdx int) (int, error) {
	depth := 0
	inString := false
	for i := openIdx; i < len(s); i++ {
		c := s[i]
		if inString {
			if c == '\'' {
				if i+1 < len(s) && s[i+1] == '\'' {
					i++
					continue
				}
				inString = false
			}
			continue
		}
		switch c {
		case '\'':
			inString = true
		case '(':
			depth++
		case ')':
			depth--
			if depth == 0 {
				return i, nil
			}
		}
	}
	return 0, fmt.Errorf("unbalanced parentheses in SQL")
}

// skipOptionalAlias skips whitespace then one bare identifier at pos, e.g.
// the "U1" the Java builder appends after a union branch's closing paren.
// It does NOT consume a leading "ORDER" — that must remain available for
// root ORDER BY detection — so it returns pos unchanged when no alias (or
// only "ORDER BY") is found there.
func skipOptionalAlias(sql string, pos int) int {
	i := pos
	for i < len(sql) && isSpace(sql[i]) {
		i++
	}
	start := i
	for i < len(sql) && isWordChar(sql[i]) {
		i++
	}
	if i == start || strings.EqualFold(sql[start:i], "ORDER") {
		return pos
	}
	return i
}

// parseBranch parses a single SELECT ... FROM ... [WHERE ...] [GROUP BY ...]
// segment (no UNION ALL, no ORDER BY — those are handled by parseFull).
func parseBranch(branch string) (*queryJSON, error) {
	selStart, selEnd, ok := findFirstTopLevel(branch, 0, []string{"SELECT"})
	if !ok || strings.TrimSpace(branch[:selStart]) != "" {
		return nil, fmt.Errorf("SQL must start with SELECT")
	}

	fromStart, fromEnd, ok := findFirstTopLevel(branch, selEnd, []string{"FROM"})
	if !ok {
		return nil, fmt.Errorf(`SQL must include a FROM clause`)
	}
	selectClause := branch[selEnd:fromStart]

	whereStart, whereEnd, hasWhere := findFirstTopLevel(branch, fromEnd, []string{"WHERE"})
	groupStart, groupEnd, hasGroup := findFirstTopLevel(branch, fromEnd, []string{"GROUP", "BY"})

	fromClauseEnd := len(branch)
	var whereClause, groupByClause string

	switch {
	case hasWhere && hasGroup:
		if groupStart < whereStart {
			return nil, fmt.Errorf("unexpected clause order in SQL (GROUP BY before WHERE)")
		}
		fromClauseEnd = whereStart
		whereClause = branch[whereEnd:groupStart]
		groupByClause = branch[groupEnd:]
	case hasWhere:
		fromClauseEnd = whereStart
		whereClause = branch[whereEnd:]
	case hasGroup:
		fromClauseEnd = groupStart
		groupByClause = branch[groupEnd:]
	}
	fromClause := branch[fromEnd:fromClauseEnd]

	q := &queryJSON{
		Select: splitTopLevelComma(selectClause),
		From:   splitFrom(fromClause),
	}
	if hasWhere {
		where, err := splitWhere(whereClause)
		if err != nil {
			return nil, err
		}
		q.Where = where
	}
	if hasGroup {
		q.GroupBy = splitTopLevelComma(groupByClause)
	}
	return q, nil
}

// splitFrom splits a FROM clause body on top-level join keywords. The text
// before the first join, and anything after a top-level comma inside a join
// segment, are plain tables (comma-separated, as before).
func splitFrom(s string) []fromItem {
	patterns := make([][]string, len(joinPatterns))
	for i, p := range joinPatterns {
		patterns[i] = p.words
	}

	// 1. Cut the clause at every top-level join keyword.
	type segment struct {
		key  string
		text string
	}
	var segments []segment
	key, last := "", 0
	for {
		start, end, idx, found := findFirstTopLevelAny(s, last, patterns)
		if !found {
			break
		}
		segments = append(segments, segment{key, s[last:start]})
		key, last = joinPatterns[idx].key, end
	}
	segments = append(segments, segment{key, s[last:]})

	// 2. Turn every segment into from items.
	var items []fromItem
	for _, seg := range segments {
		parts := splitTopLevelComma(seg.text)
		if seg.key != "" && len(parts) > 0 {
			items = append(items, parseJoin(seg.key, parts[0]))
			parts = parts[1:]
		}
		for _, p := range parts {
			items = append(items, fromItem{Table: p})
		}
	}
	return items
}

// parseJoin splits "<table> [[AS] alias] [ON cond AND cond]" into a join
// item. ON is split on top-level AND; when it has a top-level OR it can't
// be AND-joined, so it's kept as one condition wrapped in parens.
func parseJoin(key, s string) fromItem {
	item := fromItem{JoinKey: key}
	target := s
	if onStart, onEnd, found := findFirstTopLevel(s, 0, []string{"ON"}); found {
		target = s[:onStart]
		on := normalizeWhitespace(s[onEnd:])
		if _, _, hasOr := findFirstTopLevel(on, 0, []string{"OR"}); hasOr {
			item.On = []string{"(" + on + ")"}
		} else {
			item.On = splitTopLevelWord(on, "AND")
		}
	}
	item.Table, item.Alias = splitAlias(normalizeWhitespace(target))
	return item
}

// splitAlias separates a trailing bare-identifier alias ("T a", "T AS a")
// from the table. Anything else (a single name, "T USING (x)", a subquery
// without alias) stays whole in table.
func splitAlias(s string) (table, alias string) {
	i := len(s)
	for i > 0 && isWordChar(s[i-1]) {
		i--
	}
	if i == len(s) || i == 0 || !isSpace(s[i-1]) {
		return s, ""
	}
	table = strings.TrimSpace(s[:i])
	if n := len(table); n > 3 && strings.EqualFold(table[n-3:], " AS") {
		table = strings.TrimSpace(table[:n-3])
	}
	return table, s[i:]
}

// splitWhere splits a WHERE clause body on top-level AND. Top-level OR is
// rejected outright — the query JSON model only supports AND-joined
// conditions, matching the forward jsonquerytosql tool. (An OR fully inside
// one already-grouped condition, e.g. "(x = 1 OR x = 2)", is fine — it's at
// depth > 0, so it isn't "top-level".)
//
// Each resulting condition has its enclosing parens stripped, if fully
// wrapped in one — both because our own forward tool now wraps every
// condition (to fix an operator-precedence bug: joining with a bare AND
// silently changed meaning for a condition containing OR), and because the
// Java query-JSON builder in tesabiz-utils-r2 has always wrapped conditions
// the same way ("where (a) and (b)"). Stripping keeps the JSON output clean
// and round-trippable through jsonquerytosql, which re-adds the wrapping.
func splitWhere(s string) ([]string, error) {
	if _, _, found := findFirstTopLevel(s, 0, []string{"OR"}); found {
		return nil, fmt.Errorf("OR is not supported in WHERE — only AND-joined conditions")
	}
	conditions := splitTopLevelWord(s, "AND")
	for i, c := range conditions {
		conditions[i] = stripWrappingParens(c)
	}
	return conditions, nil
}

// stripWrappingParens removes one or more layers of parens that wrap the
// entire trimmed string (not just a leading/trailing paren that belongs to
// a function call or a sub-expression), tracking string literals so a
// quoted paren doesn't confuse depth tracking.
func stripWrappingParens(s string) string {
	s = strings.TrimSpace(s)
	for len(s) >= 2 && s[0] == '(' {
		depth := 0
		inString := false
		closeAt := -1
		for i := 0; i < len(s); i++ {
			c := s[i]
			if inString {
				if c == '\'' {
					if i+1 < len(s) && s[i+1] == '\'' {
						i++
						continue
					}
					inString = false
				}
				continue
			}
			switch c {
			case '\'':
				inString = true
			case '(':
				depth++
			case ')':
				depth--
				if depth == 0 {
					closeAt = i
				}
			}
			if closeAt != -1 {
				break
			}
		}
		if closeAt != len(s)-1 {
			break
		}
		s = strings.TrimSpace(s[1 : len(s)-1])
	}
	return s
}

// --- low-level scanning helpers -------------------------------------------

func isWordChar(b byte) bool {
	return b == '_' || (b >= '0' && b <= '9') || (b >= 'a' && b <= 'z') || (b >= 'A' && b <= 'Z')
}

func isSpace(b byte) bool {
	return b == ' ' || b == '\t' || b == '\n' || b == '\r'
}

func skipSpace(s string, pos int) int {
	for pos < len(s) && isSpace(s[pos]) {
		pos++
	}
	return pos
}

// matchKeywordAt tries to match the given sequence of words (e.g.
// ["GROUP", "BY"]) starting at pos, case-insensitively, requiring at least
// one whitespace run between words and a non-word character (or
// start/end-of-string) on both sides of the whole match. Returns the end
// position and true on success.
func matchKeywordAt(s string, pos int, words []string) (int, bool) {
	if pos > 0 && isWordChar(s[pos-1]) {
		return 0, false
	}
	i := pos
	for wi, w := range words {
		if wi > 0 {
			start := i
			for i < len(s) && isSpace(s[i]) {
				i++
			}
			if i == start {
				return 0, false
			}
		}
		if i+len(w) > len(s) || !strings.EqualFold(s[i:i+len(w)], w) {
			return 0, false
		}
		i += len(w)
	}
	if i < len(s) && isWordChar(s[i]) {
		return 0, false
	}
	return i, true
}

// matchKeywordAtSkippingSpace skips leading whitespace from pos, then tries
// to match word (e.g. "ALL" right after a "UNION" keyword).
func matchKeywordAtSkippingSpace(s string, pos int, word string) (start, end int, ok bool) {
	i := pos
	for i < len(s) && isSpace(s[i]) {
		i++
	}
	if i == pos {
		return 0, 0, false
	}
	end, matched := matchKeywordAt(s, i, []string{word})
	if !matched {
		return 0, 0, false
	}
	return i, end, true
}

// findFirstTopLevel scans s from startPos, tracking paren depth and string
// literals starting fresh at depth 0 (valid because startPos is always
// right after a previously-found top-level boundary), and returns the
// first position where words matches at depth 0.
func findFirstTopLevel(s string, startPos int, words []string) (start, end int, found bool) {
	start, end, _, found = findFirstTopLevelAny(s, startPos, [][]string{words})
	return start, end, found
}

// findFirstTopLevelAny is findFirstTopLevel for several keyword sequences:
// at each depth-0 position it tries every sequence in order and returns
// the first match plus the index of the sequence that matched.
func findFirstTopLevelAny(s string, startPos int, patterns [][]string) (start, end, idx int, found bool) {
	depth := 0
	inString := false
	for i := startPos; i < len(s); i++ {
		c := s[i]
		if inString {
			if c == '\'' {
				if i+1 < len(s) && s[i+1] == '\'' {
					i++
					continue
				}
				inString = false
			}
			continue
		}
		switch c {
		case '\'':
			inString = true
			continue
		case '(':
			depth++
			continue
		case ')':
			depth--
			continue
		}
		if depth == 0 {
			for pi, words := range patterns {
				if end, ok := matchKeywordAt(s, i, words); ok {
					return i, end, pi, true
				}
			}
		}
	}
	return 0, 0, 0, false
}

// splitTopLevelComma splits s on top-level commas (depth 0, outside string
// literals), trimming and whitespace-normalizing each fragment.
func splitTopLevelComma(s string) []string {
	var parts []string
	depth := 0
	inString := false
	last := 0
	for i := 0; i < len(s); i++ {
		c := s[i]
		if inString {
			if c == '\'' {
				if i+1 < len(s) && s[i+1] == '\'' {
					i++
					continue
				}
				inString = false
			}
			continue
		}
		switch c {
		case '\'':
			inString = true
		case '(':
			depth++
		case ')':
			depth--
		case ',':
			if depth == 0 {
				parts = append(parts, normalizeWhitespace(s[last:i]))
				last = i + 1
			}
		}
	}
	parts = append(parts, normalizeWhitespace(s[last:]))
	return filterEmpty(parts)
}

// splitTopLevelWord splits s on top-level occurrences of word (e.g. "AND"),
// trimming and whitespace-normalizing each fragment.
func splitTopLevelWord(s string, word string) []string {
	var parts []string
	last, pos := 0, 0
	for {
		start, end, found := findFirstTopLevel(s, pos, []string{word})
		if !found {
			break
		}
		parts = append(parts, normalizeWhitespace(s[last:start]))
		last = end
		pos = end
	}
	parts = append(parts, normalizeWhitespace(s[last:]))
	return filterEmpty(parts)
}

func normalizeWhitespace(s string) string {
	return strings.Join(strings.Fields(s), " ")
}

func filterEmpty(parts []string) []string {
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}
