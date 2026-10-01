// Package xmlfmt formats and validates XML. It registers itself with the
// tools registry so the frontend can invoke it via RunTool("xmlfmt", ...).
package xmlfmt

import (
	"errors"
	"fmt"

	"github.com/beevik/etree"

	"kitaro-toolkit/internal/tools"
)

func init() {
	tools.Register(&tools.Tool{ID: "xmlfmt", Name: "XML Formatter", Run: Run})
}

// Run validates the input XML and returns it pretty-printed (default) or
// minified, depending on opts["mode"] ("pretty" or "minify").
func Run(input string, opts map[string]string) (string, error) {
	if input == "" {
		return "", errors.New("input is empty")
	}

	doc := etree.NewDocument()
	doc.ReadSettings.PreserveCData = true
	if err := doc.ReadFromString(input); err != nil {
		return "", fmt.Errorf("invalid XML: %w", err)
	}

	if opts["mode"] == "minify" {
		doc.Indent(etree.NoIndent)
	} else {
		doc.Indent(2)
	}

	out, err := doc.WriteToString()
	if err != nil {
		return "", err
	}
	return out, nil
}
