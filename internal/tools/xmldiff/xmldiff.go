// Package xmldiff compares two XML documents and reports line-level
// differences against their canonical (pretty-printed, attribute-sorted)
// form. There is no mature Go library for XML diffing, so this canonicalizes
// both documents with etree and then runs a line-level LCS diff (golcs,
// already pulled in as gojsondiff's dependency) — the same approach a text
// diff viewer uses, which lets the frontend render the whole document with
// changed lines highlighted, like a VS Code diff view. It registers itself
// as a diff tool so the frontend can invoke it via
// RunDiffTool("xmldiff", left, right).
package xmldiff

import (
	"fmt"
	"sort"
	"strings"

	"github.com/beevik/etree"
	lcs "github.com/yudai/golcs"

	"kitaro-rq/internal/tools"
)

func init() {
	tools.RegisterDiff(&tools.DiffTool{ID: "xmldiff", Name: "XML Diff", Run: Run})
}

// Run returns the right-hand document's canonical XML with each line
// prefixed by a marker: "+" added, "-" removed, " " unchanged (context).
// Attribute order never counts as a difference, since it carries no
// semantic meaning in XML — both documents are canonicalized (attributes
// sorted alphabetically) before diffing.
func Run(left, right string) (string, error) {
	leftLines, err := canonicalLines(left)
	if err != nil {
		return "", fmt.Errorf("invalid left XML: %w", err)
	}
	rightLines, err := canonicalLines(right)
	if err != nil {
		return "", fmt.Errorf("invalid right XML: %w", err)
	}

	diffLines := lineDiff(leftLines, rightLines)

	hasDiff := false
	for _, l := range diffLines {
		if l[0] != ' ' {
			hasDiff = true
			break
		}
	}
	if !hasDiff {
		return "No differences.", nil
	}
	return strings.Join(diffLines, "\n"), nil
}

func canonicalLines(xmlStr string) ([]string, error) {
	doc := etree.NewDocument()
	doc.ReadSettings.PreserveCData = true
	if err := doc.ReadFromString(xmlStr); err != nil {
		return nil, err
	}
	sortAttrs(doc.Root())
	doc.Indent(2)

	out, err := doc.WriteToString()
	if err != nil {
		return nil, err
	}
	return strings.Split(strings.TrimRight(out, "\n"), "\n"), nil
}

func sortAttrs(e *etree.Element) {
	sort.Slice(e.Attr, func(i, j int) bool { return e.Attr[i].Key < e.Attr[j].Key })
	for _, c := range e.ChildElements() {
		sortAttrs(c)
	}
}

// lineDiff produces a unified diff between left and right using their
// longest common subsequence: lines in the LCS are context, gaps on the
// left are removed, gaps on the right are added.
func lineDiff(left, right []string) []string {
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
