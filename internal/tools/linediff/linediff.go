// Package linediff produces a unified, line-level diff between two slices of
// lines using their longest common subsequence (golcs). It is shared by the
// text-based comparators (XML Diff, SQL Diff), which canonicalize their
// inputs into lines first and then delegate the actual diffing here.
package linediff

import (
	lcs "github.com/yudai/golcs"
)

// Diff produces a unified diff between left and right using their longest
// common subsequence: lines in the LCS are context (" " prefix), gaps on the
// left are removed ("-" prefix), gaps on the right are added ("+" prefix).
func Diff(left, right []string) []string {
	leftVals := make([]interface{}, len(left))
	for i, s := range left {
		leftVals[i] = s
	}
	rightVals := make([]interface{}, len(right))
	for i, s := range right {
		rightVals[i] = s
	}

	pairs := lcs.New(leftVals, rightVals).IndexPairs()

	var out []string
	li, ri := 0, 0
	for _, p := range pairs {
		for li < p.Left {
			out = append(out, "-"+left[li])
			li++
		}
		for ri < p.Right {
			out = append(out, "+"+right[ri])
			ri++
		}
		out = append(out, " "+left[li])
		li++
		ri++
	}
	for li < len(left) {
		out = append(out, "-"+left[li])
		li++
	}
	for ri < len(right) {
		out = append(out, "+"+right[ri])
		ri++
	}
	return out
}

// HasChanges reports whether a Diff result contains any added or removed
// line (i.e. anything other than context).
func HasChanges(diffLines []string) bool {
	for _, l := range diffLines {
		if l[0] != ' ' {
			return true
		}
	}
	return false
}
