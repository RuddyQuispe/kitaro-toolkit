import { describe, expect, it } from 'vitest';
import { filterRows, parseCsv } from './csv';

describe('parseCsv', () => {
    it('parses a simple header + rows', () => {
        const { headers, rows } = parseCsv('a,b,c\n1,2,3\n4,5,6\n');
        expect(headers).toEqual(['a', 'b', 'c']);
        expect(rows).toEqual([
            ['1', '2', '3'],
            ['4', '5', '6'],
        ]);
    });

    it('handles no trailing newline on the last row', () => {
        const { headers, rows } = parseCsv('a,b\n1,2');
        expect(headers).toEqual(['a', 'b']);
        expect(rows).toEqual([['1', '2']]);
    });

    it('handles a quoted field containing a comma', () => {
        const { rows } = parseCsv('a,b\n"1,2",3\n');
        expect(rows).toEqual([['1,2', '3']]);
    });

    it('handles a quoted field containing an escaped quote', () => {
        const { rows } = parseCsv('a\n"say ""hi"""\n');
        expect(rows).toEqual([['say "hi"']]);
    });

    it('handles a quoted field containing an embedded newline', () => {
        const { rows } = parseCsv('a,b\n"line1\nline2",x\n');
        expect(rows).toEqual([['line1\nline2', 'x']]);
    });

    it('normalizes CRLF line endings', () => {
        const { headers, rows } = parseCsv('a,b\r\n1,2\r\n');
        expect(headers).toEqual(['a', 'b']);
        expect(rows).toEqual([['1', '2']]);
    });

    it('returns empty headers and rows for empty input', () => {
        const { headers, rows } = parseCsv('');
        expect(headers).toEqual([]);
        expect(rows).toEqual([]);
    });

    it('handles a header-only CSV (no data rows)', () => {
        const { headers, rows } = parseCsv('a,b,c\n');
        expect(headers).toEqual(['a', 'b', 'c']);
        expect(rows).toEqual([]);
    });
});

describe('filterRows', () => {
    const headers = ['name', 'city'];
    const rows = [
        ['Ana', 'La Paz'],
        ['Bruno', 'Cochabamba'],
        ['Carla', 'La Paz'],
    ];

    it('returns all rows when every filter is empty', () => {
        expect(filterRows(headers, rows, ['', ''])).toEqual(rows);
    });

    it('filters by a single column, case-insensitively', () => {
        expect(filterRows(headers, rows, ['', 'la paz'])).toEqual([
            ['Ana', 'La Paz'],
            ['Carla', 'La Paz'],
        ]);
    });

    it('applies multiple column filters with AND semantics', () => {
        expect(filterRows(headers, rows, ['a', 'la paz'])).toEqual([
            ['Ana', 'La Paz'],
            ['Carla', 'La Paz'],
        ]);
    });

    it('matches partial substrings', () => {
        expect(filterRows(headers, rows, ['', 'coch'])).toEqual([['Bruno', 'Cochabamba']]);
    });

    it('returns no rows when nothing matches', () => {
        expect(filterRows(headers, rows, ['zzz', ''])).toEqual([]);
    });
});
