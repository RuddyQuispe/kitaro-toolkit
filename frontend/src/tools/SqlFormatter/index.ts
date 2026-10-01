import { format, type SqlLanguage } from 'sql-formatter';
import type { ToolDef } from '../registry';
import { createCodeEditor } from '../../components/codeEditor';

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
            <div id="sqlfmt-input" class="tool-input"></div>
            <div class="tool-actions">
                <select id="sqlfmt-dialect">${options}</select>
                <button id="sqlfmt-run" class="btn">Format</button>
            </div>
            <div id="sqlfmt-output" class="tool-output"></div>
            <p id="sqlfmt-error" class="tool-error"></p>
        </div>
    `;

    const input = createCodeEditor(container.querySelector<HTMLElement>('#sqlfmt-input')!, { lang: 'sql', placeholder: 'Paste SQL here...' });
    const dialect = container.querySelector<HTMLSelectElement>('#sqlfmt-dialect')!;
    const output = createCodeEditor(container.querySelector<HTMLElement>('#sqlfmt-output')!, { lang: 'sql', readOnly: true });
    const error = container.querySelector<HTMLElement>('#sqlfmt-error')!;

    container.querySelector('#sqlfmt-run')!.addEventListener('click', () => {
        error.textContent = '';
        output.setValue('');
        try {
            output.setValue(format(input.getValue(), { language: dialect.value as SqlLanguage }));
        } catch (err) {
            error.textContent = String(err);
        }
    });

    return () => {
        input.destroy();
        output.destroy();
    };
}

export const sqlFormatterTool: ToolDef = { id: 'sqlfmt', label: 'SQL Formatter', mount };
