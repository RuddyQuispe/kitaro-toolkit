package jsondiff

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
			name:       "identical objects have no diff",
			left:       `{"a":1,"b":2}`,
			right:      `{"a":1,"b":2}`,
			wantNoDiff: true,
		},
		{
			name:        "added key",
			left:        `{"a":1}`,
			right:       `{"a":1,"b":2}`,
			wantContain: []string{`"b"`},
		},
		{
			name:        "removed key",
			left:        `{"a":1,"b":2}`,
			right:       `{"a":1}`,
			wantContain: []string{`"b"`},
		},
		{
			name:        "modified value",
			left:        `{"a":1}`,
			right:       `{"a":2}`,
			wantContain: []string{"1", "2"},
		},
		{
			name:    "malformed left json errors",
			left:    `{"a":}`,
			right:   `{"a":1}`,
			wantErr: true,
		},
		{
			name:    "malformed right json errors",
			left:    `{"a":1}`,
			right:   `{"a":}`,
			wantErr: true,
		},
		{
			name:       "identical top-level arrays have no diff",
			left:       `[1,2,3]`,
			right:      `[1,2,3]`,
			wantNoDiff: true,
		},
		{
			name:        "modified top-level array",
			left:        `[{"id":1},{"id":2}]`,
			right:       `[{"id":1},{"id":3}]`,
			wantContain: []string{"2", "3"},
		},
		{
			name:    "comparing an object with an array errors",
			left:    `{"a":1}`,
			right:   `[1,2,3]`,
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
