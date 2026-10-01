package sqldiff

import (
	"strings"
	"testing"

	"kitaro-toolkit/internal/tools"
)

func TestRun(t *testing.T) {
	tests := []struct {
		name        string
		left, right string
		wantNoDiff  bool
		wantContain []string
	}{
		{
			name:       "identical queries have no diff",
			left:       "SELECT\n  a,\n  b\nFROM\n  t",
			right:      "SELECT\n  a,\n  b\nFROM\n  t",
			wantNoDiff: true,
		},
		{
			name:       "line ending and trailing whitespace differences are ignored",
			left:       "SELECT\n  a\nFROM\n  t\n",
			right:      "SELECT  \r\n  a\r\nFROM\r\n  t",
			wantNoDiff: true,
		},
		{
			name:        "changed column shows old and new lines",
			left:        "SELECT\n  a,\n  b\nFROM\n  t",
			right:       "SELECT\n  a,\n  c\nFROM\n  t",
			wantContain: []string{" SELECT", "-  b", "+  c", " FROM"},
		},
		{
			name:        "added where clause",
			left:        "SELECT\n  a\nFROM\n  t",
			right:       "SELECT\n  a\nFROM\n  t\nWHERE\n  a = 1",
			wantContain: []string{"+WHERE", "+  a = 1"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Run(tt.left, tt.right)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if tt.wantNoDiff {
				if got != "No differences." {
					t.Errorf("expected 'No differences.', got %q", got)
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

func TestRegistered(t *testing.T) {
	d := tools.GetDiff("sqldiff")
	if d == nil {
		t.Fatal("expected sqldiff to be registered as a diff tool")
	}
	if d.Name != "SQL Diff" {
		t.Errorf("expected name 'SQL Diff', got %q", d.Name)
	}
}
