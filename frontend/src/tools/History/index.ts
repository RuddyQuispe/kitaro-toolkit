import './history.css';
import { ClearHistory, DeleteHistoryEntry, GetHistoryEntry, HistoryLimit, ListHistory } from '../../../wailsjs/go/main/App';
import { history } from '../../../wailsjs/go/models';
import type { ToolContext, ToolDef } from '../registry';
import { createCodeEditor, type CodeEditor, type EditorLang } from '../../components/codeEditor';

// Syntax for the read-only detail panes, per tool. Diff tools use `input`
// for both sides; their stored output is a plain unified diff.
const LANGS: Record<string, { input: EditorLang; output: EditorLang }> = {
    jsonfmt: { input: 'json', output: 'json' },
    xmlfmt: { input: 'xml', output: 'xml' },
    jsondiff: { input: 'json', output: 'text' },
    xmldiff: { input: 'xml', output: 'text' },
    sqldiff: { input: 'sql', output: 'text' },
    jsonquerytosql: { input: 'json', output: 'sql' },
    sqltojsonquery: { input: 'sql', output: 'json' },
    jsontocsv: { input: 'json', output: 'text' },
};

function formatTimestamp(value: string): string {
    const date = new Date(value);
    return Number.isNaN(date.getTime()) ? value : date.toLocaleString();
}

function mount(container: HTMLElement, ctx: ToolContext) {
    container.innerHTML = `
        <div class="tool-panel history-panel">
            <div class="tool-actions">
                <button id="history-refresh" class="btn btn-secondary">Refresh</button>
                <button id="history-clear" class="btn btn-secondary">Clear all</button>
                <span id="history-count" class="tool-hint"></span>
            </div>
            <div id="history-list" class="history-list"></div>
            <p id="history-error" class="tool-error"></p>
        </div>
    `;

    const list = container.querySelector<HTMLElement>('#history-list')!;
    const count = container.querySelector<HTMLElement>('#history-count')!;
    const error = container.querySelector<HTMLElement>('#history-error')!;

    // Cap comes from the backend (history.MaxEntries), the single source of truth.
    let limit = 0;

    // Read-only editors of expanded entries, keyed by entry id.
    const openDetails = new Map<string, CodeEditor[]>();

    function closeDetail(id: string) {
        openDetails.get(id)?.forEach((editor) => editor.destroy());
        openDetails.delete(id);
    }

    function closeAllDetails() {
        [...openDetails.keys()].forEach(closeDetail);
    }

    async function load() {
        error.textContent = '';
        try {
            if (!limit) limit = await HistoryLimit();
            const entries = await ListHistory();
            render(entries ?? []);
        } catch (err) {
            error.textContent = String(err);
        }
    }

    function render(entries: history.Entry[]) {
        closeAllDetails();
        count.textContent = `${entries.length}/${limit}`;

        if (entries.length === 0) {
            list.innerHTML = '<p class="empty-state">No history yet — run or compare something first.</p>';
            return;
        }

        list.innerHTML = '';
        for (const entry of entries) {
            const item = document.createElement('div');
            item.className = 'history-item';

            const header = document.createElement('div');
            header.className = 'history-item-header';

            const title = document.createElement('span');
            title.className = 'history-item-title';
            title.textContent = `${entry.toolName} · ${entry.kind === 'diff' ? 'compare' : 'format'}`;

            const time = document.createElement('span');
            time.className = 'history-item-time';
            time.textContent = formatTimestamp(entry.createdAt);

            const deleteBtn = document.createElement('button');
            deleteBtn.className = 'btn btn-secondary history-item-delete';
            deleteBtn.textContent = 'Delete';
            deleteBtn.addEventListener('click', async () => {
                try {
                    await DeleteHistoryEntry(entry.id);
                    await load();
                } catch (err) {
                    error.textContent = String(err);
                }
            });

            const viewBtn = document.createElement('button');
            viewBtn.className = 'btn btn-secondary history-item-btn';
            viewBtn.textContent = 'View';

            const detail = document.createElement('div');
            detail.className = 'history-item-detail';
            detail.hidden = true;

            viewBtn.addEventListener('click', async () => {
                if (!detail.hidden) {
                    closeDetail(entry.id);
                    detail.innerHTML = '';
                    detail.hidden = true;
                    viewBtn.textContent = 'View';
                    return;
                }
                viewBtn.disabled = true;
                try {
                    const full = await GetHistoryEntry(entry.id);
                    if (!item.isConnected) return; // list re-rendered meanwhile
                    showDetail(detail, full);
                    viewBtn.textContent = 'Hide';
                } catch (err) {
                    error.textContent = String(err);
                } finally {
                    viewBtn.disabled = false;
                }
            });

            header.append(title, time, viewBtn);

            if (ctx.hasTool(entry.toolId)) {
                const openBtn = document.createElement('button');
                openBtn.className = 'btn btn-secondary history-item-btn';
                openBtn.textContent = `Open in ${entry.toolName}`;
                openBtn.addEventListener('click', async () => {
                    openBtn.disabled = true;
                    try {
                        ctx.navigate(entry.toolId, await GetHistoryEntry(entry.id));
                    } catch (err) {
                        error.textContent = String(err);
                        openBtn.disabled = false;
                    }
                });
                header.append(openBtn);
            }

            header.append(deleteBtn);

            const body = document.createElement('pre');
            body.className = 'history-item-body';
            body.textContent =
                entry.kind === 'diff'
                    ? `LEFT:  ${entry.leftPreview ?? ''}\nRIGHT: ${entry.rightPreview ?? ''}`
                    : `INPUT:  ${entry.inputPreview ?? ''}\nOUTPUT: ${entry.outputPreview ?? ''}`;

            item.append(header, body, detail);
            list.appendChild(item);
        }
    }

    function showDetail(detail: HTMLElement, entry: history.Entry) {
        closeDetail(entry.id);
        const langs = LANGS[entry.toolId] ?? { input: 'text', output: 'text' };
        const panes: [string, string, EditorLang][] =
            entry.kind === 'diff'
                ? [
                      ['Left', entry.left ?? '', langs.input],
                      ['Right', entry.right ?? '', langs.input],
                      ['Output', entry.output ?? '', langs.output],
                  ]
                : [
                      ['Input', entry.input ?? '', langs.input],
                      ['Output', entry.output ?? '', langs.output],
                  ];

        detail.innerHTML = '';
        const editors: CodeEditor[] = [];
        for (const [label, text, lang] of panes) {
            const pane = document.createElement('div');
            pane.className = 'history-detail-pane';
            const heading = document.createElement('span');
            heading.className = 'history-detail-label';
            heading.textContent = label;
            const host = document.createElement('div');
            host.className = 'history-detail-editor';
            pane.append(heading, host);
            detail.appendChild(pane);

            const editor = createCodeEditor(host, { lang, readOnly: true });
            editor.setValue(text);
            editors.push(editor);
        }
        openDetails.set(entry.id, editors);
        detail.hidden = false;
    }

    container.querySelector('#history-refresh')!.addEventListener('click', load);
    container.querySelector('#history-clear')!.addEventListener('click', async () => {
        try {
            await ClearHistory();
            await load();
        } catch (err) {
            error.textContent = String(err);
        }
    });

    load();

    return closeAllDetails;
}

export const historyTool: ToolDef = { id: 'history', label: 'History', mount };
