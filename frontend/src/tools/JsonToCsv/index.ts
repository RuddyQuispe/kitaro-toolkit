import { RunTool, SaveFile } from '../../../wailsjs/go/main/App';
import type { ToolContext, ToolDef } from '../registry';
import { createCodeEditor } from '../../components/codeEditor';
import { filterRows, parseCsv } from './csv';
import { HistoryCursor, type Snapshot } from '../historyCursor';

function escapeHtml(s: string): string {
    return s
        .replace(/&/g, '&amp;')
        .replace(/</g, '&lt;')
        .replace(/>/g, '&gt;')
        .replace(/"/g, '&quot;');
}

// Rows are appended to the DOM in pages: a 20k-row table built in one go
// (table-layout: auto + nowrap) makes WebKit measure every cell, and every
// later relayout (button label change, filter keystroke) pays that cost
// again, freezing the UI.
const PAGE_SIZE = 500;
const FILTER_DEBOUNCE_MS = 200;

function rowsHtml(rows: string[][]): string {
    return rows
        .map((row) => `<tr>${row.map((cell) => `<td>${escapeHtml(cell)}</td>`).join('')}</tr>`)
        .join('');
}

function mount(container: HTMLElement, ctx: ToolContext) {
    container.innerHTML = `
        <div class="tool-panel">
            <div id="jsontocsv-input" class="tool-input"></div>
            <div class="tool-actions">
                <button id="jsontocsv-run" class="btn">Convert to CSV</button>
                <button id="jsontocsv-back" class="btn btn-secondary history-nav-btn" title="Previous requirement">◀</button>
                <button id="jsontocsv-forward" class="btn btn-secondary history-nav-btn" title="Next requirement">▶</button>
                <button id="jsontocsv-copy" class="btn btn-secondary" disabled>Copy</button>
                <button id="jsontocsv-save" class="btn btn-secondary" disabled>Save CSV</button>
                <span id="jsontocsv-history-status" class="history-nav-status"></span>
                <span id="jsontocsv-rowcount" class="tool-hint"></span>
            </div>
            <div id="jsontocsv-table-wrap" class="table-wrap" hidden>
                <table class="data-table">
                    <thead>
                        <tr id="jsontocsv-header-row"></tr>
                        <tr id="jsontocsv-filter-row"></tr>
                    </thead>
                    <tbody id="jsontocsv-body"></tbody>
                </table>
                <button id="jsontocsv-more" class="btn btn-secondary table-more" hidden>Show more rows</button>
            </div>
            <p id="jsontocsv-error" class="tool-error"></p>
        </div>
    `;

    const input = createCodeEditor(container.querySelector<HTMLElement>('#jsontocsv-input')!, { lang: 'json', placeholder: 'Paste JSON (array of objects, or a single object) here...' });
    const error = container.querySelector<HTMLElement>('#jsontocsv-error')!;
    const copyBtn = container.querySelector<HTMLButtonElement>('#jsontocsv-copy')!;
    const saveBtn = container.querySelector<HTMLButtonElement>('#jsontocsv-save')!;
    const rowCount = container.querySelector<HTMLElement>('#jsontocsv-rowcount')!;
    const tableWrap = container.querySelector<HTMLElement>('#jsontocsv-table-wrap')!;
    const headerRow = container.querySelector<HTMLElement>('#jsontocsv-header-row')!;
    const filterRow = container.querySelector<HTMLElement>('#jsontocsv-filter-row')!;
    const body = container.querySelector<HTMLElement>('#jsontocsv-body')!;
    const moreBtn = container.querySelector<HTMLButtonElement>('#jsontocsv-more')!;
    const backBtn = container.querySelector<HTMLButtonElement>('#jsontocsv-back')!;
    const forwardBtn = container.querySelector<HTMLButtonElement>('#jsontocsv-forward')!;
    const statusEl = container.querySelector<HTMLElement>('#jsontocsv-history-status')!;

    let rawCsv = '';
    let headers: string[] = [];
    let allRows: string[][] = [];
    let filters: string[] = [];
    let filtered: string[][] = [];
    let shown = 0;
    let filterTimer: ReturnType<typeof setTimeout> | undefined;

    const cursor = new HistoryCursor('jsontocsv');

    function updateNav() {
        backBtn.disabled = !cursor.canGoBack();
        forwardBtn.disabled = !cursor.canGoForward();
        statusEl.textContent = cursor.statusLabel();
    }

    cursor.refresh().then(() => {
        if (ctx.restore) cursor.anchorAt(ctx.restore.id);
        updateNav();
    });

    function loadResult(csv: string) {
        rawCsv = csv;
        const parsed = parseCsv(rawCsv);
        headers = parsed.headers;
        allRows = parsed.rows;
        renderTable();
        copyBtn.disabled = false;
        saveBtn.disabled = false;
    }

    function updateRowInfo() {
        const visible = shown < filtered.length ? ` (showing ${shown})` : '';
        rowCount.textContent = `${filtered.length} of ${allRows.length} rows${visible}`;
        moreBtn.hidden = shown >= filtered.length;
    }

    function appendPage() {
        const next = filtered.slice(shown, shown + PAGE_SIZE);
        body.insertAdjacentHTML('beforeend', rowsHtml(next));
        shown += next.length;
        updateRowInfo();
    }

    function renderBody() {
        filtered = filterRows(headers, allRows, filters);
        shown = 0;
        body.innerHTML = '';
        appendPage();
    }

    moreBtn.addEventListener('click', appendPage);

    function renderTable() {
        headerRow.innerHTML = headers.map((h) => `<th>${escapeHtml(h)}</th>`).join('');
        filterRow.innerHTML = headers
            .map((_, i) => `<th><input type="text" class="filter-input" data-col="${i}" placeholder="Filter..." /></th>`)
            .join('');
        filters = headers.map(() => '');
        tableWrap.hidden = false;
        renderBody();
    }

    filterRow.addEventListener('input', (e) => {
        const target = e.target as HTMLInputElement;
        const col = Number(target.dataset.col);
        if (Number.isNaN(col)) return;
        filters[col] = target.value;
        clearTimeout(filterTimer);
        filterTimer = setTimeout(renderBody, FILTER_DEBOUNCE_MS);
    });

    container.querySelector('#jsontocsv-run')!.addEventListener('click', async () => {
        error.textContent = '';
        rowCount.textContent = '';
        tableWrap.hidden = true;
        headerRow.innerHTML = '';
        filterRow.innerHTML = '';
        body.innerHTML = '';
        copyBtn.disabled = true;
        saveBtn.disabled = true;
        try {
            loadResult(await RunTool('jsontocsv', input.getValue(), {}));
            cursor.resetToPresent();
            await cursor.refresh();
            updateNav();
        } catch (err) {
            error.textContent = String(err);
        }
    });

    // Loads a saved run (Back/Forward or opened from History) without
    // re-running it.
    function applySnapshot(snap: Snapshot) {
        input.setValue(snap.input ?? '');
        loadResult(snap.output ?? '');
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

    saveBtn.addEventListener('click', async () => {
        try {
            const path = await SaveFile('result.csv', rawCsv);
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

    copyBtn.addEventListener('click', async () => {
        try {
            await navigator.clipboard.writeText(rawCsv);
            const original = copyBtn.textContent;
            copyBtn.textContent = 'Copied!';
            setTimeout(() => {
                copyBtn.textContent = original;
            }, 1200);
        } catch (err) {
            error.textContent = `Could not copy to clipboard: ${String(err)}`;
        }
    });

    if (ctx.restore) applySnapshot(ctx.restore);

    return () => {
        input.destroy();
    };
}

export const jsonToCsvTool: ToolDef = { id: 'jsontocsv', label: 'JSON to CSV', mount };
