import { format, type SqlLanguage } from 'sql-formatter';
import type { ToolDef } from '../registry';

// sql-formatter's dialect keys: Oracle is "plsql" (not "oracle"), SQL Server
// is "transactsql" (aliased from "tsql"). Verified against the installed
// package's dialectNameMap — there is no "oracle" key.
const DIALECTS: Record<string, SqlLanguage> = {
    Oracle: 'plsql',
    'SQL Server': 'transactsql',
};

function mount(container: HTMLElement) {
    const options = Object.keys(DIALECTS)
        .map((label) => `<option value="${DIALECTS[label]}">${label}</option>`)
        .join('');

    container.innerHTML = `
        <div class="tool-panel">
            <textarea id="sqlfmt-input" class="tool-input" placeholder="Paste SQL here..."></textarea>
            <div class="tool-actions">
                <select id="sqlfmt-dialect">${options}</select>
                <button id="sqlfmt-run" class="btn">Format</button>
            </div>
            <pre id="sqlfmt-output" class="tool-output"></pre>
            <p id="sqlfmt-error" class="tool-error"></p>
        </div>
    `;

    const input = container.querySelector<HTMLTextAreaElement>('#sqlfmt-input')!;
    const dialect = container.querySelector<HTMLSelectElement>('#sqlfmt-dialect')!;
    const output = container.querySelector<HTMLElement>('#sqlfmt-output')!;
    const error = container.querySelector<HTMLElement>('#sqlfmt-error')!;

    container.querySelector('#sqlfmt-run')!.addEventListener('click', () => {
        error.textContent = '';
        output.textContent = '';
        try {
            output.textContent = format(input.value, { language: dialect.value as SqlLanguage });
        } catch (err) {
            error.textContent = String(err);
        }
    });
}

export const sqlFormatterTool: ToolDef = { id: 'sqlfmt', label: 'SQL Formatter', mount };
