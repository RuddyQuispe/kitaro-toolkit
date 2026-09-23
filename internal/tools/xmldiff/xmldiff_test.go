package xmldiff

import (
	"strings"
	"testing"
)

func TestRun(t *testing.T) {
	tests := []struct {
		name        string
		left, right string
		wantErr     bool
		wantNoDiff  bool
		wantContain []string
	}{
		{
			name:       "identical documents have no diff",
			left:       `<root><a>1</a></root>`,
			right:      `<root><a>1</a></root>`,
			wantNoDiff: true,
		},
		{
			name:       "attribute order does not matter",
			left:       `<item id="1" name="x"/>`,
			right:      `<item name="x" id="1"/>`,
			wantNoDiff: true,
		},
		{
			name:        "changed text content",
			left:        `<root><a>1</a></root>`,
			right:       `<root><a>2</a></root>`,
			wantContain: []string{"/root/a", "1", "2"},
		},
		{
			name:        "added child element",
			left:        `<root><a>1</a></root>`,
			right:       `<root><a>1</a><b>2</b></root>`,
			wantContain: []string{"added", "/root/b"},
		},
		{
			name:        "removed child element",
			left:        `<root><a>1</a><b>2</b></root>`,
			right:       `<root><a>1</a></root>`,
			wantContain: []string{"removed", "/root/b"},
		},
		{
			name:        "changed attribute value",
			left:        `<item id="1"/>`,
			right:       `<item id="2"/>`,
			wantContain: []string{"/item@id", "1", "2"},
		},
		{
			name:    "malformed left xml errors",
			left:    `<root><a>1</root>`,
			right:   `<root><a>1</a></root>`,
			wantErr: true,
		},
		{
			name:    "malformed right xml errors",
			left:    `<root><a>1</a></root>`,
			right:   `<root><a>1</root>`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Run(tt.left, tt.right)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error, got none (output: %q)", got)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if tt.wantNoDiff {
				if !strings.Contains(strings.ToLower(got), "no differ") {
					t.Errorf("expected 'no differences' message, got %q", got)
				}
				return
			}
			for _, s := range tt.wantContain {
				if !strings.Contains(got, s) {
					t.Errorf("expected output to contain %q, got %q", s, got)
				}
			}
		})
	}
}
