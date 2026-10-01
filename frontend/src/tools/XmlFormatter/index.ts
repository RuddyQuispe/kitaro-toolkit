import { RunTool } from '../../../wailsjs/go/main/App';
import type { ToolContext, ToolDef } from '../registry';
import { createCodeEditor } from '../../components/codeEditor';
import { HistoryCursor, restoreSelect, type Snapshot } from '../historyCursor';

function mount(container: HTMLElement, ctx: ToolContext) {
    container.innerHTML = `
        <div class="tool-panel">
            <div id="xmlfmt-input" class="tool-input"></div>
            <div class="tool-actions">
                <select id="xmlfmt-mode">
                    <option value="pretty">Pretty</option>
                    <option value="minify">Minify</option>
                </select>
                <button id="xmlfmt-run" class="btn">Format</button>
                <button id="xmlfmt-back" class="btn btn-secondary history-nav-btn" title="Previous requirement">◀</button>
                <button id="xmlfmt-forward" class="btn btn-secondary history-nav-btn" title="Next requirement">▶</button>
                <span id="xmlfmt-history-status" class="history-nav-status tool-hint"></span>
            </div>
            <div id="xmlfmt-output" class="tool-output"></div>
            <p id="xmlfmt-error" class="tool-error"></p>
        </div>
    `;

    const input = createCodeEditor(container.querySelector<HTMLElement>('#xmlfmt-input')!, { lang: 'xml', placeholder: 'Paste XML here...' });
    const mode = container.querySelector<HTMLSelectElement>('#xmlfmt-mode')!;
    const output = createCodeEditor(container.querySelector<HTMLElement>('#xmlfmt-output')!, { lang: 'xml', readOnly: true });
    const error = container.querySelector<HTMLElement>('#xmlfmt-error')!;
    const backBtn = container.querySelector<HTMLButtonElement>('#xmlfmt-back')!;
    const forwardBtn = container.querySelector<HTMLButtonElement>('#xmlfmt-forward')!;
    const statusEl = container.querySelector<HTMLElement>('#xmlfmt-history-status')!;

    const cursor = new HistoryCursor('xmlfmt');

    function updateNav() {
        backBtn.disabled = !cursor.canGoBack();
        forwardBtn.disabled = !cursor.canGoForward();
        statusEl.textContent = cursor.statusLabel();
    }

    cursor.refresh().then(() => {
        if (ctx.restore) cursor.anchorAt(ctx.restore.id);
        updateNav();
    });

    container.querySelector('#xmlfmt-run')!.addEventListener('click', async () => {
        error.textContent = '';
        output.setValue('');
        try {
            output.setValue(await RunTool('xmlfmt', input.getValue(), { mode: mode.value }));
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

export const xmlFormatterTool: ToolDef = { id: 'xmlfmt', label: 'XML Formatter', mount };
