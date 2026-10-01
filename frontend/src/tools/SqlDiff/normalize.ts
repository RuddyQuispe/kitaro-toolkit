import { format, type SqlLanguage } from 'sql-formatter';

/**
 * Pretty-prints sql in a canonical layout (one clause/column per line,
 * upper-case keywords) so the backend line diff only reports real changes,
 * not whitespace or keyword-casing noise. Throws on unparsable SQL.
 */
export function normalizeSql(sql: string, language: SqlLanguage): string {
    if (sql.trim() === '') return '';
    return format(sql, { language, keywordCase: 'upper' });
}
