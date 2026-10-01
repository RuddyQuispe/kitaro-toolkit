import { RunTool } from '../../../wailsjs/go/main/App';
import type { ToolContext, ToolDef } from '../registry';
import { createCodeEditor } from '../../components/codeEditor';
import { HistoryCursor, restoreSelect, type Snapshot } from '../historyCursor';

function mount(container: HTMLElement, ctx: ToolContext) {
    container.innerHTML = `
        <div class="tool-panel">
            <div id="jsonfmt-input" class="tool-input"></div>
            <div class="tool-actions">
                <select id="jsonfmt-mode">
                    <option value="pretty">Pretty</option>
                    <option value="minify">Minify</option>
                </select>
                <button id="jsonfmt-run" class="btn">Format</button>
                <button id="jsonfmt-back" class="btn btn-secondary history-nav-btn" title="Previous requirement">◀</button>
                <button id="jsonfmt-forward" class="btn btn-secondary history-nav-btn" title="Next requirement">▶</button>
                <span id="jsonfmt-history-status" class="history-nav-status tool-hint"></span>
            </div>
            <div id="jsonfmt-output" class="tool-output"></div>
            <p id="jsonfmt-error" class="tool-error"></p>
        </div>
    `;

    const input = createCodeEditor(container.querySelector<HTMLElement>('#jsonfmt-input')!, { lang: 'json', placeholder: 'Paste JSON here...' });
    const mode = container.querySelector<HTMLSelectElement>('#jsonfmt-mode')!;
    const output = createCodeEditor(container.querySelector<HTMLElement>('#jsonfmt-output')!, { lang: 'json', readOnly: true });
    const error = container.querySelector<HTMLElement>('#jsonfmt-error')!;
    const backBtn = container.querySelector<HTMLButtonElement>('#jsonfmt-back')!;
    const forwardBtn = container.querySelector<HTMLButtonElement>('#jsonfmt-forward')!;
    const statusEl = container.querySelector<HTMLElement>('#jsonfmt-history-status')!;

    const cursor = new HistoryCursor('jsonfmt');

    function updateNav() {
        backBtn.disabled = !cursor.canGoBack();
        forwardBtn.disabled = !cursor.canGoForward();
        statusEl.textContent = cursor.statusLabel();
    }

    cursor.refresh().then(() => {
        if (ctx.restore) cursor.anchorAt(ctx.restore.id);
        updateNav();
    });

    container.querySelector('#jsonfmt-run')!.addEventListener('click', async () => {
        error.textContent = '';
        output.setValue('');
        try {
            output.setValue(await RunTool('jsonfmt', input.getValue(), { mode: mode.value }));
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
        restoreSelect(mode, snap.options?.mode);
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

    if (ctx.restore) applySnapshot(ctx.restore);

    return () => {
        input.destroy();
        output.destroy();
    };
}

export const jsonFormatterTool: ToolDef = { id: 'jsonfmt', label: 'JSON Formatter', mount };
