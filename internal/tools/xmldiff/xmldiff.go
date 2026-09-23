// Package xmldiff compares two XML documents and reports structural
// differences. There is no mature Go library for XML diffing, so this
// walks two etree trees recursively and reports differences by XPath-like
// location. It registers itself as a diff tool so the frontend can invoke
// it via RunDiffTool("xmldiff", left, right).
package xmldiff

import (
	"fmt"
	"sort"
	"strings"

	"github.com/beevik/etree"

	"kitaro-rq/internal/tools"
)

func init() {
	tools.RegisterDiff(&tools.DiffTool{ID: "xmldiff", Name: "XML Diff", Run: Run})
}

// Run compares the root elements of left and right XML documents and
// returns a list of differences (one per line), or "No differences." if
// they are structurally equal. Attribute order never counts as a
// difference, since it carries no semantic meaning in XML.
func Run(left, right string) (string, error) {
	leftDoc := etree.NewDocument()
	if err := leftDoc.ReadFromString(left); err != nil {
		return "", fmt.Errorf("invalid left XML: %w", err)
	}
	rightDoc := etree.NewDocument()
	if err := rightDoc.ReadFromString(right); err != nil {
		return "", fmt.Errorf("invalid right XML: %w", err)
	}

	var diffs []string
	diffElements(leftDoc.Root(), rightDoc.Root(), "", &diffs)

	if len(diffs) == 0 {
		return "No differences.", nil
	}
	return strings.Join(diffs, "\n"), nil
}

func diffElements(a, b *etree.Element, path string, diffs *[]string) {
	p := path + "/" + a.Tag
	if a.Tag != b.Tag {
		*diffs = append(*diffs, fmt.Sprintf("changed tag at %s: %q -> %q", path, a.Tag, b.Tag))
		return
	}

	diffAttributes(a, b, p, diffs)

	aText, bText := strings.TrimSpace(a.Text()), strings.TrimSpace(b.Text())
	if aText != bText {
		*diffs = append(*diffs, fmt.Sprintf("changed text at %s: %q -> %q", p, aText, bText))
	}

	aChildren, bChildren := a.ChildElements(), b.ChildElements()
	n := len(aChildren)
	if len(bChildren) < n {
		n = len(bChildren)
	}
	for i := 0; i < n; i++ {
		diffElements(aChildren[i], bChildren[i], p, diffs)
	}
	for i := n; i < len(aChildren); i++ {
		*diffs = append(*diffs, fmt.Sprintf("removed %s/%s", p, aChildren[i].Tag))
	}
	for i := n; i < len(bChildren); i++ {
		*diffs = append(*diffs, fmt.Sprintf("added %s/%s", p, bChildren[i].Tag))
	}
}

func diffAttributes(a, b *etree.Element, path string, diffs *[]string) {
	aAttrs := attrMap(a)
	bAttrs := attrMap(b)

	keys := make(map[string]bool)
	for k := range aAttrs {
		keys[k] = true
	}
	for k := range bAttrs {
		keys[k] = true
	}
	sorted := make([]string, 0, len(keys))
	for k := range keys {
		sorted = append(sorted, k)
	}
	sort.Strings(sorted)

	for _, k := range sorted {
		av, aok := aAttrs[k]
		bv, bok := bAttrs[k]
		switch {
		case !aok:
			*diffs = append(*diffs, fmt.Sprintf("added attribute %s@%s: %q", path, k, bv))
		case !bok:
			*diffs = append(*diffs, fmt.Sprintf("removed attribute %s@%s: %q", path, k, av))
		case av != bv:
			*diffs = append(*diffs, fmt.Sprintf("changed %s@%s: %q -> %q", path, k, av, bv))
		}
	}
}

func attrMap(e *etree.Element) map[string]string {
	m := make(map[string]string, len(e.Attr))
	for _, a := range e.Attr {
		m[a.Key] = a.Value
	}
	return m
}
