package xmlfmt

import (
	"strings"
	"testing"
)

func TestRun(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		opts    map[string]string
		want    string
		wantErr bool
	}{
		{
			name:  "pretty print default",
			input: `<root><a>1</a><b>2</b></root>`,
			opts:  nil,
			want:  "<root>\n  <a>1</a>\n  <b>2</b>\n</root>",
		},
		{
			name:  "pretty print with attributes",
			input: `<root><item id="1" name="x"/></root>`,
			opts:  nil,
			want:  "<root>\n  <item id=\"1\" name=\"x\"/>\n</root>",
		},
		{
			name:  "minify",
			input: "<root>\n  <a>1</a>\n  <b>2</b>\n</root>",
			opts:  map[string]string{"mode": "minify"},
			want:  `<root><a>1</a><b>2</b></root>`,
		},
		{
			name:  "preserves cdata",
			input: `<root><![CDATA[some <raw> text]]></root>`,
			opts:  nil,
			want:  "<root><![CDATA[some <raw> text]]></root>",
		},
		{
			name:  "preserves comments",
			input: `<root><!-- a comment --><a>1</a></root>`,
			opts:  nil,
			want:  "<root>\n  <!-- a comment -->\n  <a>1</a>\n</root>",
		},
		{
			name:  "namespaces preserved",
			input: `<root xmlns:ns="http://example.com"><ns:item>1</ns:item></root>`,
			opts:  nil,
			want:  "<root xmlns:ns=\"http://example.com\">\n  <ns:item>1</ns:item>\n</root>",
		},
		{
			name:    "unclosed tag returns error",
			input:   `<root><a>1</root>`,
			wantErr: true,
		},
		{
			name:    "empty input returns error",
			input:   "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Run(tt.input, tt.opts)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error, got none (output: %q)", got)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if strings.TrimSpace(got) != strings.TrimSpace(tt.want) {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}
