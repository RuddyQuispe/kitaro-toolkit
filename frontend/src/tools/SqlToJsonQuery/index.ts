import { RunTool, SaveFile } from '../../../wailsjs/go/main/App';
import type { ToolContext, ToolDef } from '../registry';
import { createCodeEditor } from '../../components/codeEditor';
import { HistoryCursor, type Snapshot } from '../historyCursor';

function mount(container: HTMLElement, ctx: ToolContext) {
    container.innerHTML = `
        <div class="tool-panel">
            <div id="sqltojsonquery-input" class="tool-input"></div>
            <div class="tool-actions">
                <button id="sqltojsonquery-run" class="btn">Convert to JSON Query</button>
                <button id="sqltojsonquery-back" class="btn btn-secondary history-nav-btn" title="Previous requirement">◀</button>
                <button id="sqltojsonquery-forward" class="btn btn-secondary history-nav-btn" title="Next requirement">▶</button>
                <button id="sqltojsonquery-copy" class="btn btn-secondary" disabled>Copy</button>
                <button id="sqltojsonquery-save" class="btn btn-secondary" disabled>Save</button>
                <span id="sqltojsonquery-history-status" class="history-nav-status tool-hint"></span>
            </div>
            <div id="sqltojsonquery-output" class="tool-output"></div>
            <p id="sqltojsonquery-error" class="tool-error"></p>
        </div>
    `;

    const input = createCodeEditor(container.querySelector<HTMLElement>('#sqltojsonquery-input')!, { lang: 'sql', placeholder: 'Paste SQL here (SELECT/FROM/WHERE/GROUP BY/ORDER BY, optional UNION ALL chain)...' });
    const output = createCodeEditor(container.querySelector<HTMLElement>('#sqltojsonquery-output')!, { lang: 'json', readOnly: true });
    const error = container.querySelector<HTMLElement>('#sqltojsonquery-error')!;
    const copyBtn = container.querySelector<HTMLButtonElement>('#sqltojsonquery-copy')!;
    const saveBtn = container.querySelector<HTMLButtonElement>('#sqltojsonquery-save')!;
    const backBtn = container.querySelector<HTMLButtonElement>('#sqltojsonquery-back')!;
    const forwardBtn = container.querySelector<HTMLButtonElement>('#sqltojsonquery-forward')!;
    const statusEl = container.querySelector<HTMLElement>('#sqltojsonquery-history-status')!;

    const cursor = new HistoryCursor('sqltojsonquery');

    function updateNav() {
        backBtn.disabled = !cursor.canGoBack();
        forwardBtn.disabled = !cursor.canGoForward();
        statusEl.textContent = cursor.statusLabel();
    }

    cursor.refresh().then(() => {
        if (ctx.restore) cursor.anchorAt(ctx.restore.id);
        updateNav();
    });

    container.querySelector('#sqltojsonquery-run')!.addEventListener('click', async () => {
        error.textContent = '';
        output.setValue('');
        copyBtn.disabled = true;
        saveBtn.disabled = true;
        try {
            output.setValue(await RunTool('sqltojsonquery', input.getValue(), {}));
            copyBtn.disabled = false;
            saveBtn.disabled = false;
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
        output.setValue(snap.output ?? '');
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
            const path = await SaveFile('query.json', output.getValue());
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

export const sqlToJsonQueryTool: ToolDef = { id: 'sqltojsonquery', label: 'SQL to JSON Query', mount };
