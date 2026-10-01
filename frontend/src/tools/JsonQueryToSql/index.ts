import { format, type SqlLanguage } from 'sql-formatter';
import { RunTool, SaveFile } from '../../../wailsjs/go/main/App';
import type { ToolContext, ToolDef } from '../registry';
import { createCodeEditor } from '../../components/codeEditor';
import { HistoryCursor, restoreSelect, type Snapshot } from '../historyCursor';

// Same dialect keys as SqlFormatter: Oracle is "plsql", SQL Server is
// "transactsql" — see that tool for the sql-formatter dialectNameMap note.
const DIALECTS: Record<string, SqlLanguage> = {
    Oracle: 'plsql',
    'SQL Server': 'transactsql',
};

function mount(container: HTMLElement, ctx: ToolContext) {
    const options = Object.keys(DIALECTS)
        .map((label) => `<option value="${DIALECTS[label]}">${label}</option>`)
        .join('');

    container.innerHTML = `
        <div class="tool-panel">
            <div id="jsonquerytosql-input" class="tool-input"></div>
            <div class="tool-actions">
                <select id="jsonquerytosql-dialect">${options}</select>
                <button id="jsonquerytosql-run" class="btn">Convert to SQL</button>
                <button id="jsonquerytosql-back" class="btn btn-secondary history-nav-btn" title="Previous requirement">◀</button>
                <button id="jsonquerytosql-forward" class="btn btn-secondary history-nav-btn" title="Next requirement">▶</button>
                <button id="jsonquerytosql-copy" class="btn btn-secondary" disabled>Copy</button>
                <button id="jsonquerytosql-save" class="btn btn-secondary" disabled>Save</button>
                <span id="jsonquerytosql-history-status" class="history-nav-status tool-hint"></span>
            </div>
            <div id="jsonquerytosql-output" class="tool-output"></div>
            <p id="jsonquerytosql-error" class="tool-error"></p>
        </div>
    `;

    const input = createCodeEditor(container.querySelector<HTMLElement>('#jsonquerytosql-input')!, { lang: 'json', placeholder: 'Paste query JSON (select/from/where/groupBy/orderBy/unionAll) here...' });
    const dialect = container.querySelector<HTMLSelectElement>('#jsonquerytosql-dialect')!;
    const output = createCodeEditor(container.querySelector<HTMLElement>('#jsonquerytosql-output')!, { lang: 'sql', readOnly: true });
    const error = container.querySelector<HTMLElement>('#jsonquerytosql-error')!;
    const copyBtn = container.querySelector<HTMLButtonElement>('#jsonquerytosql-copy')!;
    const saveBtn = container.querySelector<HTMLButtonElement>('#jsonquerytosql-save')!;
    const backBtn = container.querySelector<HTMLButtonElement>('#jsonquerytosql-back')!;
    const forwardBtn = container.querySelector<HTMLButtonElement>('#jsonquerytosql-forward')!;
    const statusEl = container.querySelector<HTMLElement>('#jsonquerytosql-history-status')!;

    const cursor = new HistoryCursor('jsonquerytosql');

    function updateNav() {
        backBtn.disabled = !cursor.canGoBack();
        forwardBtn.disabled = !cursor.canGoForward();
        statusEl.textContent = cursor.statusLabel();
    }

    cursor.refresh().then(() => {
        if (ctx.restore) cursor.anchorAt(ctx.restore.id);
        updateNav();
    });

    container.querySelector('#jsonquerytosql-run')!.addEventListener('click', async () => {
        error.textContent = '';
        output.setValue('');
        copyBtn.disabled = true;
        saveBtn.disabled = true;
        try {
            const rawSql = await RunTool('jsonquerytosql', input.getValue(), { dialect: dialect.value });
            output.setValue(format(rawSql, { language: dialect.value as SqlLanguage }));
            copyBtn.disabled = false;
            saveBtn.disabled = false;
            cursor.resetToPresent();
            await cursor.refresh();
            updateNav();
        } catch (err) {
            error.textContent = String(err);
        }
    });

    // Entry.output always holds the raw SQL from before client-side
    // formatting (see the run handler above), so every restored snapshot —
    // present or past — needs the same format() pass, with the dialect the
    // run used when it was saved (older entries keep the current one).
    function applySnapshot(snap: Snapshot) {
        restoreSelect(dialect, snap.options?.dialect);
        input.setValue(snap.input ?? '');
        output.setValue(format(snap.output ?? '', { language: dialect.value as SqlLanguage }));
        copyBtn.disabled = false;
        saveBtn.disabled = false;
    }

    async function browse(step: () => Promise<Snapshot | null>) {
        error.textContent = '';
        try {
            const snap = await step();
            if (snap) applySnapshot(snap);
        } catch (err) {
            error.textContent = String(err);
        }
        updateNav();
    }

    backBtn.addEventListener('click', () => browse(() => cursor.back()));
    forwardBtn.addEventListener('click', () => browse(() => cursor.forward()));

    copyBtn.addEventListener('click', async () => {
        try {
            await navigator.clipboard.writeText(output.getValue());
            const original = copyBtn.textContent;
            copyBtn.textContent = 'Copied!';
            setTimeout(() => {
                copyBtn.textContent = original;
            }, 1200);
        } catch (err) {
            error.textContent = `Could not copy to clipboard: ${String(err)}`;
        }
    });

    saveBtn.addEventListener('click', async () => {
        try {
            const path = await SaveFile('query.sql', output.getValue());
            if (!path) return; // user cancelled the dialog
            const original = saveBtn.textContent;
            saveBtn.textContent = 'Saved!';
            setTimeout(() => {
                saveBtn.textContent = original;
            }, 1200);
        } catch (err) {
            error.textContent = `Could not save file: ${String(err)}`;
        }
    });

    if (ctx.restore) applySnapshot(ctx.restore);

    return () => {
        input.destroy();
        output.destroy();
    };
}

export const jsonQueryToSqlTool: ToolDef = { id: 'jsonquerytosql', label: 'JSON Query to SQL', mount };
