// Package sqldiff compares two SQL statements line by line. The frontend
// normalizes both sides with sql-formatter (per dialect) before calling it,
// so layout noise is already gone; this package only smooths line endings
// and trailing whitespace, then runs the shared LCS line diff. It registers
// itself as a diff tool so the frontend can invoke it via
// RunDiffTool("sqldiff", left, right).
package sqldiff

import (
	"strings"

	"kitaro-toolkit/internal/tools"
	"kitaro-toolkit/internal/tools/linediff"
)

func init() {
	tools.RegisterDiff(&tools.DiffTool{ID: "sqldiff", Name: "SQL Diff", Run: Run})
}

// Run returns the diff of the two SQL texts with each line prefixed by a
// marker: "+" added, "-" removed, " " unchanged (context), or
// "No differences." when both sides match.
func Run(left, right string) (string, error) {
	diffLines := linediff.Diff(normalizedLines(left), normalizedLines(right))
	if !linediff.HasChanges(diffLines) {
		return "No differences.", nil
	}
	return strings.Join(diffLines, "\n"), nil
}

// normalizedLines splits sql into lines, dropping CRLF/trailing whitespace
// differences and trailing blank lines, which carry no meaning in SQL.
func normalizedLines(sql string) []string {
	lines := strings.Split(strings.ReplaceAll(sql, "\r\n", "\n"), "\n")
	for i, l := range lines {
		lines[i] = strings.TrimRight(l, " \t\r")
	}
	for len(lines) > 0 && lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}
	return lines
}
