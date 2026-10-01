package linediff

import (
	"reflect"
	"testing"
)

func TestDiff(t *testing.T) {
	tests := []struct {
		name        string
		left, right []string
		want        []string
	}{
		{
			name:  "identical lines are all context",
			left:  []string{"a", "b"},
			right: []string{"a", "b"},
			want:  []string{" a", " b"},
		},
		{
			name:  "changed line shows removal then addition",
			left:  []string{"a", "b", "c"},
			right: []string{"a", "x", "c"},
			want:  []string{" a", "-b", "+x", " c"},
		},
		{
			name:  "trailing additions",
			left:  []string{"a"},
			right: []string{"a", "b"},
			want:  []string{" a", "+b"},
		},
		{
			name:  "trailing removals",
			left:  []string{"a", "b"},
			right: []string{"a"},
			want:  []string{" a", "-b"},
		},
		{
			name:  "empty left is all additions",
			left:  nil,
			right: []string{"a", "b"},
			want:  []string{"+a", "+b"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Diff(tt.left, tt.right)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Diff(%q, %q) = %q, want %q", tt.left, tt.right, got, tt.want)
			}
		})
	}
}

func TestHasChanges(t *testing.T) {
	if HasChanges([]string{" a", " b"}) {
		t.Error("expected no changes for context-only lines")
	}
	if !HasChanges([]string{" a", "+b"}) {
		t.Error("expected changes when an added line is present")
	}
	if HasChanges(nil) {
		t.Error("expected no changes for empty diff")
	}
}
