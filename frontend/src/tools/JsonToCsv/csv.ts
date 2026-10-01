// Minimal RFC 4180-aware CSV parser (quoted fields, embedded commas/quotes/
// newlines, "" as an escaped quote, CRLF or LF line endings). Matches the
// exact output shape produced by Go's encoding/csv on the backend, so no
// CSV-parsing dependency is needed just to render a table view.

export interface ParsedCsv {
    headers: string[];
    rows: string[][];
}

export function parseCsv(text: string): ParsedCsv {
    const records = parseCsvRecords(text);
    if (records.length === 0) {
        return { headers: [], rows: [] };
    }
    const [headers, ...rows] = records;
    return { headers, rows };
}

function parseCsvRecords(text: string): string[][] {
    const records: string[][] = [];
    let record: string[] = [];
    let field = '';
    let inQuotes = false;

    for (let i = 0; i < text.length; i++) {
        const c = text[i];

        if (inQuotes) {
            if (c === '"') {
                if (text[i + 1] === '"') {
                    field += '"';
                    i++;
                } else {
                    inQuotes = false;
                }
            } else {
                field += c;
            }
            continue;
        }

        switch (c) {
            case '"':
                inQuotes = true;
                break;
            case ',':
                record.push(field);
                field = '';
                break;
            case '\r':
                break;
            case '\n':
                record.push(field);
                records.push(record);
                record = [];
                field = '';
                break;
            default:
                field += c;
        }
    }

    if (field !== '' || record.length > 0) {
        record.push(field);
        records.push(record);
    }

    return records;
}

// filterRows keeps rows where every non-empty filter (one per column, same
// index as headers) matches that column's value as a case-insensitive
// substring. Empty filters are ignored (match everything).
export function filterRows(headers: string[], rows: string[][], filters: string[]): string[][] {
    const needles = filters.map((f) => f.trim().toLowerCase());
    return rows.filter((row) =>
        headers.every((_, colIdx) => {
            const needle = needles[colIdx];
            if (!needle) return true;
            return (row[colIdx] ?? '').toLowerCase().includes(needle);
        }),
    );
}
