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
			name:  "orderBy as a plain string (legacy format)",
			input: `{"select":["u.USER_LOGIN"],"from":["M_USER_MOBILE um","T_USER u"],"where":["um.USER_ID = u.USER_ID","u.USER_LOGIN = :userLogin"],"orderBy":"u.USER_LOGIN ASC","other":"oracle"}`,
			wantOrder: []string{
				"FROM M_USER_MOBILE um, T_USER u",
				"WHERE (um.USER_ID = u.USER_ID) AND (u.USER_LOGIN = :userLogin)",
				"ORDER BY u.USER_LOGIN ASC",
			},
		},
		{
			name:    "orderBy that is neither string nor array errors",
			input:   `{"select":["a"],"from":["T"],"orderBy":1}`,
			wantErr: true,
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
			name:  "cte renders WITH before the main select",
			input: `{"cte":{"name":"P","definition":{"select":["x"],"from":["T1"]}},"select":["x"],"from":["P"]}`,
			wantOrder: []string{
				"WITH P AS (SELECT x FROM T1)",
				" SELECT x",
				"FROM P",
			},
		},
		{
			name:        "innerJoin with alias and several on conditions",
			input:       `{"select":["a"],"from":["T1 a",{"innerJoin":"T2","name":"b","on":["a.id = b.id","b.x = 1"]}]}`,
			wantContain: []string{"FROM T1 a INNER JOIN T2 b ON a.id = b.id AND b.x = 1"},
		},
		{
			name:        "leftJoin without alias nor on",
			input:       `{"select":["a"],"from":["T1",{"leftJoin":"T2"}]}`,
			wantContain: []string{"FROM T1 LEFT JOIN T2"},
		},
		{
			name: "full example with cte, joins and nested conditions",
			input: `{
				"cte":{"name":"PARTICIPANT_TOTAL","definition":{
					"select":["DISPERSION.CLIENT_PARAMETER_ID","SUM(DC.AMOUNT) AS AMOUNT"],
					"from":["COM_DISPERSION DISPERSION",
						{"innerJoin":"COM_CLIENT","name":"CLIENT","on":["CLIENT.CLIENT_ID = DISPERSION.CLIENT_ID"]}],
					"where":["DISPERSION.DISPERSION_ID = :dispersionId"],
					"groupBy":["DISPERSION.CLIENT_PARAMETER_ID"]}},
				"select":["TOTAL.AMOUNT"],
				"from":["PARTICIPANT_TOTAL TOTAL",
					{"leftJoin":"HISTORY_PARTICIPANT","name":"HISTORY_PARTICIPANT","on":["HISTORY_PARTICIPANT.ID = TOTAL.ID","(HISTORY_PARTICIPANT.END_DATE IS NULL OR HISTORY_PARTICIPANT.END_DATE > TOTAL.DATE_REGISTER)"]}],
				"orderBy":["TOTAL.ID ASC NULLS LAST"],
				"other":"oracle"
			}`,
			wantOrder: []string{
				"WITH PARTICIPANT_TOTAL AS (SELECT",
				"INNER JOIN COM_CLIENT CLIENT ON CLIENT.CLIENT_ID = DISPERSION.CLIENT_ID",
				"WHERE (DISPERSION.DISPERSION_ID = :dispersionId)",
				"GROUP BY DISPERSION.CLIENT_PARAMETER_ID)",
				" SELECT TOTAL.AMOUNT",
				"FROM PARTICIPANT_TOTAL TOTAL LEFT JOIN HISTORY_PARTICIPANT HISTORY_PARTICIPANT ON HISTORY_PARTICIPANT.ID = TOTAL.ID AND (HISTORY_PARTICIPANT.END_DATE IS NULL OR",
				"ORDER BY TOTAL.ID ASC NULLS LAST",
			},
		},
		{
			name:    "join object as first from item errors",
			input:   `{"select":["a"],"from":[{"innerJoin":"T2","on":["x = y"]}]}`,
			wantErr: true,
		},
		{
			name:    "join object with two join keys errors",
			input:   `{"select":["a"],"from":["T1",{"innerJoin":"T2","leftJoin":"T3"}]}`,
			wantErr: true,
		},
		{
			name:    "join object without join key errors",
			input:   `{"select":["a"],"from":["T1",{"name":"b"}]}`,
			wantErr: true,
		},
		{
			name:    "cte without name errors",
			input:   `{"cte":{"definition":{"select":["x"],"from":["T"]}},"select":["x"],"from":["P"]}`,
			wantErr: true,
		},
		{
			name:    "cte without definition errors",
			input:   `{"cte":{"name":"P"},"select":["x"],"from":["P"]}`,
			wantErr: true,
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
