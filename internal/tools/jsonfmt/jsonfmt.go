// Package jsonfmt formats and validates JSON. It registers itself with the
// tools registry so the frontend can invoke it via RunTool("jsonfmt", ...).
package jsonfmt

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"

	"kitaro-rq/internal/tools"
)

func init() {
	tools.Register(&tools.Tool{ID: "jsonfmt", Name: "JSON Formatter", Run: Run})
}

// Run validates the input JSON and returns it pretty-printed (default) or
// minified, depending on opts["mode"] ("pretty" or "minify").
// json.Indent/Compact operate on raw bytes rather than an intermediate map,
// so key order and number literals are preserved verbatim.
func Run(input string, opts map[string]string) (string, error) {
	if input == "" {
		return "", errors.New("input is empty")
	}
	if !json.Valid([]byte(input)) {
		return "", describeSyntaxError(input)
	}

	var buf bytes.Buffer
	var err error
	if opts["mode"] == "minify" {
		err = json.Compact(&buf, []byte(input))
	} else {
		err = json.Indent(&buf, []byte(input), "", "  ")
	}
	if err != nil {
		return "", err
	}
	return buf.String(), nil
}

func describeSyntaxError(input string) error {
	var v any
	err := json.Unmarshal([]byte(input), &v)
	var syntaxErr *json.SyntaxError
	if errors.As(err, &syntaxErr) {
		return fmt.Errorf("invalid JSON at byte offset %d: %s", syntaxErr.Offset, syntaxErr.Error())
	}
	return fmt.Errorf("invalid JSON: %w", err)
}
