import { describe, expect, it } from 'vitest';
import { format } from 'sql-formatter';

describe('sql-formatter dialects', () => {
    it('formats Oracle-specific syntax without breaking (NVL, ROWNUM, CONNECT BY)', () => {
        const sql =
            "select nvl(a, 0), rownum from dual connect by prior id = parent_id";
        const out = format(sql, { language: 'plsql' }).toLowerCase();
        expect(out).toContain('nvl');
        expect(out).toContain('rownum');
        expect(out).toContain('connect by');
    });

    it('formats SQL Server-specific syntax without breaking (TOP, ISNULL, brackets)', () => {
        const sql = 'select top 10 isnull([name], \'-\') from [dbo].[users]';
        const out = format(sql, { language: 'transactsql' }).toLowerCase();
        expect(out).toContain('top 10');
        expect(out).toContain('isnull');
        expect(out).toContain('[name]');
    });

    it('produces multi-line indented output for a simple query', () => {
        const out = format('select a, b from t where a = 1', { language: 'plsql' });
        expect(out.split('\n').length).toBeGreaterThan(1);
    });
});
