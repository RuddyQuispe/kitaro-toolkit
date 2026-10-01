import { describe, expect, it } from 'vitest';
import { normalizeSql } from './normalize';

describe('normalizeSql', () => {
    it('makes layout-only differences identical', () => {
        const a = normalizeSql('select a, b from t where a = 1', 'plsql');
        const b = normalizeSql('SELECT a,\n      b\n  FROM t\n WHERE a=1', 'plsql');
        expect(a).toBe(b);
    });

    it('puts each selected column on its own line so a changed column is one diff line', () => {
        const out = normalizeSql('select a, b, c from t', 'transactsql');
        const lines = out.split('\n').map((l) => l.trim());
        expect(lines).toContain('a,');
        expect(lines).toContain('b,');
        expect(lines).toContain('c');
    });

    it('returns an empty string for blank input', () => {
        expect(normalizeSql('   \n ', 'plsql')).toBe('');
    });
});
