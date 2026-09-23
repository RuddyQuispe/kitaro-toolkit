// Package jsondiff compares two JSON documents and reports structural
// differences. It registers itself as a diff tool so the frontend can
// invoke it via RunDiffTool("jsondiff", left, right).
package jsondiff

import (
	"encoding/json"
	"fmt"

	diff "github.com/yudai/gojsondiff"
	"github.com/yudai/gojsondiff/formatter"

	"kitaro-rq/internal/tools"
)

func init() {
	tools.RegisterDiff(&tools.DiffTool{ID: "jsondiff", Name: "JSON Diff", Run: Run})
}

// Run compares left and right JSON documents and returns a human-readable
// ASCII diff (lines prefixed with + / -), or "No differences." if they are
// structurally equal.
func Run(left, right string) (string, error) {
	differ := diff.New()
	d, err := differ.Compare([]byte(left), []byte(right))
	if err != nil {
		return "", fmt.Errorf("invalid JSON: %w", err)
	}

	if !d.Modified() {
		return "No differences.", nil
	}

	var leftJSON map[string]interface{}
	if err := json.Unmarshal([]byte(left), &leftJSON); err != nil {
		return "", fmt.Errorf("invalid JSON: %w", err)
	}

	f := formatter.NewAsciiFormatter(leftJSON, formatter.AsciiFormatterConfig{
		ShowArrayIndex: true,
	})
	out, err := f.Format(d)
	if err != nil {
		return "", err
	}
	return out, nil
}
