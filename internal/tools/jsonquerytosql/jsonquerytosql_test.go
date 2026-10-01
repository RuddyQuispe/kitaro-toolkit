package jsonquerytosql

import (
	"strings"
	"testing"
)

func TestRun(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		wantErr     bool
		wantContain []string
		wantOrder   []string // substrings that must each appear, in this order
	}{
		{
			name:        "simple select/from",
			input:       `{"select":["a","b"],"from":["T"]}`,
			wantContain: []string{"SELECT a, b", "FROM T"},
		},
		{
			// Each condition is wrapped in its own parentheses, matching the
			// behavior of the Java query-JSON builder in tesabiz-utils-r2
			// (QueryBuilder.whereAndHavingClause: "where (a) and (b)"). Without
			// this, a condition containing an OR (e.g. "x = 1 OR y = 2") would
			// silently change meaning once joined with AND due to operator
			// precedence.
			name:        "where conditions joined with AND, each wrapped in parens",
			input:       `{"select":["a"],"from":["T"],"where":["a = 1","b = 2"]}`,
			wantContain: []string{"WHERE (a = 1) AND (b = 2)"},
		},
		{
			name:        "multiple from tables joined with comma",
			input:       `{"select":["a"],"from":["T1","T2"]}`,
			wantContain: []string{"FROM T1, T2"},
		},
		{
			name:        "groupBy joined with comma",
			input:       `{"select":["a"],"from":["T"],"groupBy":["a","b"]}`,
			wantContain: []string{"GROUP BY a, b"},
		},
		{
			name:  "orderBy appears after the query",
			input: `{"select":["a"],"from":["T"],"orderBy":["a, b"]}`,
			wantOrder: []string{
				"SELECT a",
				"FROM T",
				"ORDER BY a, b",
			},
		},
		{
			name: "unionAll chains two selects and orderBy lands at the very end",
			input: `{
				"select":["a"],"from":["T1"],
				"unionAll":{"select":["a"],"from":["T2"]},
				"orderBy":["a"]
			}`,
			wantOrder: []string{
				"SELECT a",
				"FROM T1",
				"UNION ALL",
				"SELECT a",
				"FROM T2",
				"ORDER BY a",
			},
		},
		{
			name: "unionAll chains three selects (nested)",
			input: `{
				"select":["a"],"from":["T1"],
				"unionAll":{
					"select":["a"],"from":["T2"],
					"unionAll":{"select":["a"],"from":["T3"]}
				},
				"orderBy":["a"]
			}`,
			wantOrder: []string{
				"FROM T1",
				"UNION ALL",
				"FROM T2",
				"UNION ALL",
				"FROM T3",
				"ORDER BY a",
			},
		},
		{
			name:    "missing select errors",
			input:   `{"from":["T"]}`,
			wantErr: true,
		},
		{
			name:    "missing from errors",
			input:   `{"select":["a"]}`,
			wantErr: true,
		},
		{
			name:    "malformed json errors",
			input:   `{"select":}`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Run(tt.input, nil)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error, got none (output: %q)", got)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			for _, s := range tt.wantContain {
				if !strings.Contains(got, s) {
					t.Errorf("expected output to contain %q, got %q", s, got)
				}
			}
			if len(tt.wantOrder) > 0 {
				searchFrom := 0
				for _, s := range tt.wantOrder {
					idx := strings.Index(got[searchFrom:], s)
					if idx == -1 {
						t.Fatalf("expected output to contain %q after position %d, got %q", s, searchFrom, got)
					}
					searchFrom += idx + len(s)
				}
			}
		})
	}
}
