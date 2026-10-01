// Package jsondiff compares two JSON documents and reports structural
// differences. It registers itself as a diff tool so the frontend can
// invoke it via RunDiffTool("jsondiff", left, right).
package jsondiff

import (
	"encoding/json"
	"fmt"

	diff "github.com/yudai/gojsondiff"
	"github.com/yudai/gojsondiff/formatter"

	"kitaro-toolkit/internal/tools"
)

func init() {
	tools.RegisterDiff(&tools.DiffTool{ID: "jsondiff", Name: "JSON Diff", Run: Run})
}

// Run compares left and right JSON documents and returns a human-readable
// ASCII diff (lines prefixed with + / -), or "No differences." if they are
// structurally equal. Both top-level JSON objects and top-level JSON arrays
// are supported.
func Run(left, right string) (string, error) {
	var leftValue, rightValue interface{}
	if err := json.Unmarshal([]byte(left), &leftValue); err != nil {
		return "", fmt.Errorf("invalid JSON: %w", err)
	}
	if err := json.Unmarshal([]byte(right), &rightValue); err != nil {
		return "", fmt.Errorf("invalid JSON: %w", err)
	}

	differ := diff.New()
	var d diff.Diff
	switch leftTyped := leftValue.(type) {
	case map[string]interface{}:
		rightTyped, ok := rightValue.(map[string]interface{})
		if !ok {
			return "", fmt.Errorf("cannot compare a JSON object with a JSON %T", rightValue)
		}
		d = differ.CompareObjects(leftTyped, rightTyped)
	case []interface{}:
		rightTyped, ok := rightValue.([]interface{})
		if !ok {
			return "", fmt.Errorf("cannot compare a JSON array with a JSON %T", rightValue)
		}
		d = differ.CompareArrays(leftTyped, rightTyped)
	default:
		return "", fmt.Errorf("invalid JSON: top-level value must be an object or an array")
	}

	if !d.Modified() {
		return "No differences.", nil
	}

	f := formatter.NewAsciiFormatter(leftValue, formatter.AsciiFormatterConfig{
		ShowArrayIndex: true,
	})
	out, err := f.Format(d)
	if err != nil {
		return "", err
	}
	return out, nil
}
