// Package jsontocsv converts a JSON document into CSV. It registers itself
// with the tools registry so the frontend can invoke it via
// RunTool("jsontocsv", ...).
package jsontocsv

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"sort"

	"kitaro-rq/internal/tools"
)

func init() {
	tools.Register(&tools.Tool{ID: "jsontocsv", Name: "JSON to CSV", Run: Run})
}

// Run converts input JSON to CSV. An array of objects becomes one row per
// object; a single object becomes a one-row table. Nested objects are
// flattened with dot-notation keys (e.g. "user.address.city"). Rows may
// have different keys — the header is the alphabetically sorted union of
// all keys (map key order in Go is randomized, so this is the only way to
// get a deterministic column order), and missing values become empty cells.
func Run(input string, _ map[string]string) (string, error) {
	if input == "" {
		return "", errors.New("input is empty")
	}

	var root any
	dec := json.NewDecoder(bytes.NewReader([]byte(input)))
	dec.UseNumber()
	if err := dec.Decode(&root); err != nil {
		return "", fmt.Errorf("invalid JSON: %w", err)
	}

	rows, ok := root.([]any)
	if !ok {
		rows = []any{root}
	}
	if len(rows) == 0 {
		return "", errors.New("input has no rows to convert")
	}

	flatRows := make([]map[string]string, len(rows))
	seen := make(map[string]bool)

	for i, row := range rows {
		flat := make(map[string]string)
		flatten("", row, flat)
		flatRows[i] = flat
		for k := range flat {
			seen[k] = true
		}
	}

	headers := make([]string, 0, len(seen))
	for k := range seen {
		headers = append(headers, k)
	}
	sort.Strings(headers)

	var buf bytes.Buffer
	w := csv.NewWriter(&buf)
	if err := w.Write(headers); err != nil {
		return "", err
	}
	for _, flat := range flatRows {
		record := make([]string, len(headers))
		for i, h := range headers {
			record[i] = flat[h]
		}
		if err := w.Write(record); err != nil {
			return "", err
		}
	}
	w.Flush()
	if err := w.Error(); err != nil {
		return "", err
	}
	return buf.String(), nil
}

// flatten walks v and fills out with "prefix.key" -> stringified scalar for
// every leaf value.
func flatten(prefix string, v any, out map[string]string) {
	switch val := v.(type) {
	case map[string]any:
		for k, child := range val {
			flatten(joinKey(prefix, k), child, out)
		}
	case []any:
		for i, child := range val {
			flatten(fmt.Sprintf("%s[%d]", prefix, i), child, out)
		}
	case nil:
		out[prefix] = ""
	case json.Number:
		out[prefix] = val.String()
	case string:
		out[prefix] = val
	case bool:
		out[prefix] = fmt.Sprintf("%v", val)
	default:
		out[prefix] = fmt.Sprintf("%v", val)
	}
}

func joinKey(prefix, key string) string {
	if prefix == "" {
		return key
	}
	return prefix + "." + key
}
