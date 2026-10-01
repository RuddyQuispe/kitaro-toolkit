// Package sqltojsonquery converts a raw SQL statement into the "query JSON"
// document shape used by internal/tools/jsonquerytosql (select/from/where/
// groupBy/orderBy, with an optional recursively-nested unionAll). It is the
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
	Select   []string   `json:"select,omitempty"`
	From     []string   `json:"from,omitempty"`
	Where    []string   `json:"where,omitempty"`
	GroupBy  []string   `json:"groupBy,omitempty"`
	OrderBy  []string   `json:"orderBy,omitempty"`
	UnionAll *queryJSON `json:"unionAll,omitempty"`
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

	q, err := parseFull(sql)
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
		From:   splitTopLevelComma(fromClause),
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
			if end, ok := matchKeywordAt(s, i, words); ok {
				return i, end, true
			}
		}
	}
	return 0, 0, false
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
