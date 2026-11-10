package sqltojsonquery

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"

	jsonquerytosql "kitaro-toolkit/internal/tools/jsonquerytosql"
)

func decode(t *testing.T, s string) queryJSON {
	t.Helper()
	var q queryJSON
	if err := json.Unmarshal([]byte(s), &q); err != nil {
		t.Fatalf("invalid JSON output: %v\n%s", err, s)
	}
	return q
}

// tables returns the Table of every from item, so tests that only use
// plain tables can compare against a []string.
func tables(items []fromItem) []string {
	out := make([]string, len(items))
	for i, it := range items {
		out[i] = it.Table
	}
	return out
}

func TestRun_SimpleSelectFrom(t *testing.T) {
	out, err := Run("SELECT a, b FROM t", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	q := decode(t, out)
	if !reflect.DeepEqual(q.Select, []string{"a", "b"}) {
		t.Errorf("select = %v", q.Select)
	}
	if !reflect.DeepEqual(tables(q.From), []string{"t"}) {
		t.Errorf("from = %v", q.From)
	}
}

func TestRun_WhereJoinedWithAnd(t *testing.T) {
	out, err := Run("SELECT a FROM t WHERE x = 1 AND y = 2", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	q := decode(t, out)
	if !reflect.DeepEqual(q.Where, []string{"x = 1", "y = 2"}) {
		t.Errorf("where = %v", q.Where)
	}
}

func TestRun_MultipleFromTables_OracleOuterJoin(t *testing.T) {
	sql := `SELECT u.USER_ID, u.USER_EMAIL, tp.NAME AS ACCESS_TYPE_NAME
FROM T_USER u, T_USER_COMPANY uc, T_PROFILE tp
WHERE u.USER_ID = uc.USER_ID(+) AND u.ACCESS_TYPE = tp.PROFILE_ID AND uc.COMPANY_ID = :companyId`
	out, err := Run(sql, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	q := decode(t, out)
	wantSelect := []string{"u.USER_ID", "u.USER_EMAIL", "tp.NAME AS ACCESS_TYPE_NAME"}
	if !reflect.DeepEqual(q.Select, wantSelect) {
		t.Errorf("select = %v", q.Select)
	}
	wantFrom := []string{"T_USER u", "T_USER_COMPANY uc", "T_PROFILE tp"}
	if !reflect.DeepEqual(tables(q.From), wantFrom) {
		t.Errorf("from = %v", q.From)
	}
	wantWhere := []string{"u.USER_ID = uc.USER_ID(+)", "u.ACCESS_TYPE = tp.PROFILE_ID", "uc.COMPANY_ID = :companyId"}
	if !reflect.DeepEqual(q.Where, wantWhere) {
		t.Errorf("where = %v", q.Where)
	}
}

func TestRun_JoinChainWithNvlAndQuotedStrings(t *testing.T) {
	sql := `SELECT CONCIL_LOTE.ID_CONCILIACION_LOTE, COMPANY.COMPANY_ID, COMPANY.CODE AS COMPANY_CODE, NVL(EIFS.DOM_NAME, '-') AS EIF_NAME, ACCOUNT.EFI AS EIF
FROM CON_CONCILIACION_LOTE CONCIL_LOTE inner join T_SISTORIGEN SISORIGEN ON SISORIGEN.CLIENT_ID = CONCIL_LOTE.ID_CLIENTE AND SISORIGEN.CODE = CONCIL_LOTE.SO inner join M_CLIENT CLIENT on CONCIL_LOTE.ID_CLIENTE = CLIENT.CLIENT_ID left join T_DOMAIN EIFS ON EIFS.DOMAIN_TBZ = 'EIFS' AND EIFS.DOM_VALUE = ACCOUNT.EFI
WHERE CONCIL_LOTE.DETALLE = 'INICIO' AND CLIENT.CLIENT_ID = :clientIdentifier`
	out, err := Run(sql, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	q := decode(t, out)
	if len(q.Select) != 5 {
		t.Errorf("select count = %d, want 5: %v", len(q.Select), q.Select)
	}
	// Each top-level join becomes its own from item; the AND inside an ON
	// clause splits into separate on conditions.
	want := []fromItem{
		{Table: "CON_CONCILIACION_LOTE CONCIL_LOTE"},
		{JoinKey: "innerJoin", Table: "T_SISTORIGEN", Alias: "SISORIGEN", On: []string{"SISORIGEN.CLIENT_ID = CONCIL_LOTE.ID_CLIENTE", "SISORIGEN.CODE = CONCIL_LOTE.SO"}},
		{JoinKey: "innerJoin", Table: "M_CLIENT", Alias: "CLIENT", On: []string{"CONCIL_LOTE.ID_CLIENTE = CLIENT.CLIENT_ID"}},
		{JoinKey: "leftJoin", Table: "T_DOMAIN", Alias: "EIFS", On: []string{"EIFS.DOMAIN_TBZ = 'EIFS'", "EIFS.DOM_VALUE = ACCOUNT.EFI"}},
	}
	if !reflect.DeepEqual(q.From, want) {
		t.Errorf("from = %+v", q.From)
	}
	// AND inside the ON clauses must NOT leak into the top-level where split.
	if !reflect.DeepEqual(q.Where, []string{"CONCIL_LOTE.DETALLE = 'INICIO'", "CLIENT.CLIENT_ID = :clientIdentifier"}) {
		t.Errorf("where = %v", q.Where)
	}
}

func TestRun_GroupBy(t *testing.T) {
	out, err := Run("SELECT a, COUNT(*) FROM t GROUP BY a", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	q := decode(t, out)
	if !reflect.DeepEqual(q.Select, []string{"a", "COUNT(*)"}) {
		t.Errorf("select = %v", q.Select)
	}
	if !reflect.DeepEqual(q.GroupBy, []string{"a"}) {
		t.Errorf("groupBy = %v", q.GroupBy)
	}
}

func TestRun_OrderByAtRoot(t *testing.T) {
	out, err := Run("SELECT a, b FROM t ORDER BY a, b", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	q := decode(t, out)
	if !reflect.DeepEqual(q.OrderBy, []string{"a", "b"}) {
		t.Errorf("orderBy = %v", q.OrderBy)
	}
}

// The Java query-JSON builder in tesabiz-utils-r2 (QueryBuilder) wraps each
// WHERE/HAVING condition in its own parens: "where (a) and (b)". We must
// strip that wrapping when parsing so round-tripping through our own
// forward tool (which now also wraps, to fix an operator-precedence bug)
// produces the same clean condition text.
func TestRun_WhereWithJavaStyleWrappedParens(t *testing.T) {
	out, err := Run("SELECT a FROM t WHERE (x = 1) AND (y = 2)", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	q := decode(t, out)
	if !reflect.DeepEqual(q.Where, []string{"x = 1", "y = 2"}) {
		t.Errorf("where = %v", q.Where)
	}
}

func TestRun_WhereWithJavaStyleWrappedParens_OrInsideOneCondition(t *testing.T) {
	// An OR fully inside one already-grouped condition must NOT trip the
	// "OR is not supported" error — only a top-level OR between conditions
	// is rejected.
	out, err := Run("SELECT a FROM t WHERE (x = 1 OR x = 2) AND (y = 2)", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	q := decode(t, out)
	if !reflect.DeepEqual(q.Where, []string{"x = 1 OR x = 2", "y = 2"}) {
		t.Errorf("where = %v", q.Where)
	}
}

// The Java builder renders "A union all (B union all (C))" — only the
// right-hand branch of each union is wrapped in parens, nested to the
// right. Root ORDER BY (and an optional branch alias, e.g. "U1") sit after
// the outermost closing paren.
func TestRun_UnionAllJavaStyleParenthesizedBranches(t *testing.T) {
	sql := "SELECT a FROM t1 UNION ALL (SELECT a FROM t2 UNION ALL (SELECT a FROM t3)) order by a"
	out, err := Run(sql, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	q := decode(t, out)
	if !reflect.DeepEqual(tables(q.From), []string{"t1"}) {
		t.Errorf("root from = %v", q.From)
	}
	if q.UnionAll == nil || q.UnionAll.UnionAll == nil {
		t.Fatalf("expected a 2-level deep unionAll chain, got: %+v", q)
	}
	if !reflect.DeepEqual(tables(q.UnionAll.From), []string{"t2"}) {
		t.Errorf("branch 2 from = %v", q.UnionAll.From)
	}
	if !reflect.DeepEqual(tables(q.UnionAll.UnionAll.From), []string{"t3"}) {
		t.Errorf("branch 3 from = %v", q.UnionAll.UnionAll.From)
	}
	if !reflect.DeepEqual(q.OrderBy, []string{"a"}) {
		t.Errorf("root orderBy = %v", q.OrderBy)
	}
}

func TestRun_UnionAllJavaStyleWithBranchAlias(t *testing.T) {
	sql := `SELECT a FROM t1 UNION ALL (SELECT a FROM t2) U1`
	out, err := Run(sql, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	q := decode(t, out)
	if q.UnionAll == nil || !reflect.DeepEqual(tables(q.UnionAll.From), []string{"t2"}) {
		t.Errorf("branch from = %v", q.UnionAll)
	}
	if q.OrderBy != nil {
		t.Errorf("orderBy should be empty (alias U1 must not be mistaken for it), got %v", q.OrderBy)
	}
}

func TestRun_UnionAllTwoBranches_OrderByStaysAtRoot(t *testing.T) {
	out, err := Run("SELECT a FROM t1 UNION ALL SELECT a FROM t2 ORDER BY a", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	q := decode(t, out)
	if !reflect.DeepEqual(tables(q.From), []string{"t1"}) {
		t.Errorf("root from = %v", q.From)
	}
	if q.UnionAll == nil {
		t.Fatalf("expected unionAll, got nil")
	}
	if !reflect.DeepEqual(tables(q.UnionAll.From), []string{"t2"}) {
		t.Errorf("unionAll from = %v", q.UnionAll.From)
	}
	if q.UnionAll.OrderBy != nil {
		t.Errorf("branch orderBy should be empty, got %v", q.UnionAll.OrderBy)
	}
	if !reflect.DeepEqual(q.OrderBy, []string{"a"}) {
		t.Errorf("root orderBy = %v", q.OrderBy)
	}
}

func TestRun_UnionAllThreeBranches_NestedChain(t *testing.T) {
	sql := "SELECT a FROM t1 UNION ALL SELECT a FROM t2 UNION ALL SELECT a FROM t3"
	out, err := Run(sql, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	q := decode(t, out)
	if q.UnionAll == nil || q.UnionAll.UnionAll == nil {
		t.Fatalf("expected a 2-level deep unionAll chain, got: %+v", q)
	}
	if q.UnionAll.UnionAll.UnionAll != nil {
		t.Errorf("expected chain to stop at depth 2")
	}
	if !reflect.DeepEqual(tables(q.UnionAll.UnionAll.From), []string{"t3"}) {
		t.Errorf("deepest from = %v", q.UnionAll.UnionAll.From)
	}
}

func TestRun_Errors(t *testing.T) {
	cases := []struct {
		name        string
		sql         string
		errContains string
	}{
		{"missing select", "FROM t", "SELECT"},
		{"missing from", "SELECT a", "FROM"},
		{"where with or", "SELECT a FROM t WHERE x = 1 OR y = 2", "OR"},
		{"clauses out of order", "SELECT a FROM t GROUP BY a WHERE x = 1", "order"},
		{"unbalanced parens", "SELECT a FROM t WHERE (x = 1", "parenthes"},
		{"union without all", "SELECT a FROM t1 UNION SELECT a FROM t2", "UNION ALL"},
		{"multiple ctes", "WITH a AS (SELECT x FROM t), b AS (SELECT y FROM u) SELECT x FROM a", "one CTE"},
		{"cte without as", "WITH a (SELECT x FROM t) SELECT x FROM a", "WITH"},
		{"cte without main select", "WITH a AS (SELECT x FROM t)", "SELECT"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := Run(tc.sql, nil)
			if err == nil {
				t.Fatalf("expected error, got none")
			}
			if !strings.Contains(err.Error(), tc.errContains) {
				t.Errorf("error %q does not contain %q", err.Error(), tc.errContains)
			}
		})
	}
}

func TestRun_RoundTripWithForwardTool(t *testing.T) {
	sql := `SELECT u.USER_ID, u.USER_EMAIL
FROM T_USER u, T_PROFILE tp
WHERE u.ACCESS_TYPE = tp.PROFILE_ID AND tp.COMPANY_ID = :companyId`

	out1, err := Run(sql, nil)
	if err != nil {
		t.Fatalf("first parse failed: %v", err)
	}
	q1 := decode(t, out1)

	forwardSQL, err := jsonquerytosql.Run(out1, nil)
	if err != nil {
		t.Fatalf("forward tool rejected generated JSON: %v", err)
	}

	out2, err := Run(forwardSQL, nil)
	if err != nil {
		t.Fatalf("re-parsing forward-generated SQL failed: %v\nSQL was:\n%s", err, forwardSQL)
	}
	q2 := decode(t, out2)

	if !reflect.DeepEqual(q1, q2) {
		t.Errorf("round trip mismatch:\nq1 = %+v\nq2 = %+v", q1, q2)
	}
}

func TestRun_JoinKindsAliasesAndPlainTablesAfterJoins(t *testing.T) {
	sql := `SELECT a FROM T1 a LEFT OUTER JOIN T2 AS b ON b.id = a.id RIGHT JOIN T3 ON T3.id = a.id FULL OUTER JOIN T4 d ON (d.x IS NULL OR d.x = a.x) JOIN T5 e ON e.id = a.id OR e.id IS NULL, T6 f`
	out, err := Run(sql, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	q := decode(t, out)
	want := []fromItem{
		{Table: "T1 a"},
		{JoinKey: "leftJoin", Table: "T2", Alias: "b", On: []string{"b.id = a.id"}},
		{JoinKey: "rightJoin", Table: "T3", On: []string{"T3.id = a.id"}},
		{JoinKey: "fullJoin", Table: "T4", Alias: "d", On: []string{"(d.x IS NULL OR d.x = a.x)"}},
		// A top-level OR can't be expressed as AND-joined conditions, so the
		// whole ON stays as one condition, wrapped to keep its meaning.
		{JoinKey: "join", Table: "T5", Alias: "e", On: []string{"(e.id = a.id OR e.id IS NULL)"}},
		{Table: "T6 f"},
	}
	if !reflect.DeepEqual(q.From, want) {
		t.Errorf("from = %+v", q.From)
	}
}

func TestRun_JoinJSONShape(t *testing.T) {
	out, err := Run("SELECT a FROM T1 x INNER JOIN T2 y ON y.id = x.id", nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, `"innerJoin": "T2",`) || !strings.Contains(out, `"name": "y",`) || !strings.Contains(out, `"on": [`) {
		t.Errorf("unexpected JSON shape:\n%s", out)
	}
	if strings.Index(out, `"innerJoin"`) > strings.Index(out, `"name"`) {
		t.Errorf("join key must come before name:\n%s", out)
	}
}

func TestRun_CTE(t *testing.T) {
	sql := `with P as (select x, SUM(y) AS Y from T1 where (T1.k = :k) group by x) select P.x from P order by P.x`
	out, err := Run(sql, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	q := decode(t, out)
	if q.CTE == nil || q.CTE.Name != "P" || q.CTE.Definition == nil {
		t.Fatalf("cte = %+v", q.CTE)
	}
	d := q.CTE.Definition
	if !reflect.DeepEqual(d.Select, []string{"x", "SUM(y) AS Y"}) || !reflect.DeepEqual(d.Where, []string{"T1.k = :k"}) || !reflect.DeepEqual(d.GroupBy, []string{"x"}) {
		t.Errorf("cte definition = %+v", d)
	}
	if !reflect.DeepEqual(q.Select, []string{"P.x"}) || !reflect.DeepEqual(tables(q.From), []string{"P"}) || !reflect.DeepEqual(q.OrderBy, []string{"P.x"}) {
		t.Errorf("main query = %+v", q)
	}
	if !strings.HasPrefix(strings.TrimSpace(out), "{\n  \"cte\"") {
		t.Errorf("cte must be the first key:\n%s", out)
	}
}

func TestRun_RoundTripCTEAndJoins(t *testing.T) {
	sql := `with PARTICIPANT_TOTAL as (select D.ID, SUM(C.AMOUNT) AS AMOUNT from COM_DISPERSION D inner join COM_CLIENT CLIENT on CLIENT.CLIENT_ID = D.CLIENT_ID where (D.DISPERSION_ID = :dispersionId) group by D.ID) select TOTAL.ID, (SELECT H.P FROM CFG C JOIN HIST H ON H.ID = C.ID WHERE C.X = TOTAL.ID ORDER BY C.Y NULLS LAST FETCH FIRST 1 ROWS ONLY) AS PERCENTAGE from PARTICIPANT_TOTAL TOTAL left join HISTORY_PARTICIPANT HP on HP.ID = TOTAL.ID and HP.START_DATE <= TOTAL.D and (HP.END_DATE IS NULL OR HP.END_DATE > TOTAL.D) order by TOTAL.ID ASC NULLS LAST`

	out1, err := Run(sql, nil)
	if err != nil {
		t.Fatalf("first parse failed: %v", err)
	}
	forwardSQL, err := jsonquerytosql.Run(out1, nil)
	if err != nil {
		t.Fatalf("forward tool rejected generated JSON: %v\n%s", err, out1)
	}
	out2, err := Run(forwardSQL, nil)
	if err != nil {
		t.Fatalf("re-parsing forward-generated SQL failed: %v\nSQL was:\n%s", err, forwardSQL)
	}
	if out1 != out2 {
		t.Errorf("round trip mismatch:\nout1 = %s\nout2 = %s", out1, out2)
	}
	q := decode(t, out1)
	if len(q.From) != 2 || len(q.From[1].On) != 3 {
		t.Errorf("main from = %+v", q.From)
	}
}

func TestRun_CrossJoin(t *testing.T) {
	sql := `SELECT a FROM COM_DISPERSION D INNER JOIN COM_CLIENT_PARAMETER CP ON CP.ID = D.CP_ID CROSS JOIN CLIENT_EXTENDED cross join T3 c WHERE CLIENT_EXTENDED.CLIENT_ID = CP.CLIENT_ID`
	out, err := Run(sql, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	q := decode(t, out)
	want := []fromItem{
		{Table: "COM_DISPERSION D"},
		{JoinKey: "innerJoin", Table: "COM_CLIENT_PARAMETER", Alias: "CP", On: []string{"CP.ID = D.CP_ID"}},
		{JoinKey: "crossJoin", Table: "CLIENT_EXTENDED"},
		{JoinKey: "crossJoin", Table: "T3", Alias: "c"},
	}
	if !reflect.DeepEqual(q.From, want) {
		t.Errorf("from = %+v", q.From)
	}

	forwardSQL, err := jsonquerytosql.Run(out, nil)
	if err != nil {
		t.Fatalf("forward tool rejected generated JSON: %v\n%s", err, out)
	}
	if out2, _ := Run(forwardSQL, nil); out2 != out {
		t.Errorf("round trip mismatch:\nout1 = %s\nout2 = %s", out, out2)
	}
}
