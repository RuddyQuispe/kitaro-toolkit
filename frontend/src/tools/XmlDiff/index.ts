import { RunDiffTool } from '../../../wailsjs/go/main/App';
import type { ToolContext, ToolDef } from '../registry';
import { createCodeEditor } from '../../components/codeEditor';
import { destroyDiffView, renderDiffView } from '../../components/diffViewer';
import { renderDiffSummary } from '../../components/diffSummary';
import { HistoryCursor, type Snapshot } from '../historyCursor';

function mount(container: HTMLElement, ctx: ToolContext) {
    container.innerHTML = `
        <div class="tool-panel diff-panel">
            <div class="diff-inputs">
                <div id="xmldiff-left" class="tool-input"></div>
                <div id="xmldiff-right" class="tool-input"></div>
            </div>
            <div class="tool-actions">
                <button id="xmldiff-run" class="btn">Compare</button>
                <button id="xmldiff-back" class="btn btn-secondary history-nav-btn" title="Previous requirement">◀</button>
                <button id="xmldiff-forward" class="btn btn-secondary history-nav-btn" title="Next requirement">▶</button>
                <button id="xmldiff-copy" class="btn btn-secondary" disabled>Copy</button>
                <span id="xmldiff-history-status" class="history-nav-status tool-hint"></span>
            </div>
            <p id="xmldiff-summary" class="diff-summary"></p>
            <div id="xmldiff-viewer" class="diff-viewer"></div>
            <p id="xmldiff-error" class="tool-error"></p>
        </div>
    `;

    const left = createCodeEditor(container.querySelector<HTMLElement>('#xmldiff-left')!, { lang: 'xml', placeholder: 'Left XML...' });
    const right = createCodeEditor(container.querySelector<HTMLElement>('#xmldiff-right')!, { lang: 'xml', placeholder: 'Right XML...' });
    const summary = container.querySelector<HTMLElement>('#xmldiff-summary')!;
    const viewer = container.querySelector<HTMLElement>('#xmldiff-viewer')!;
    const error = container.querySelector<HTMLElement>('#xmldiff-error')!;
    const copyBtn = container.querySelector<HTMLButtonElement>('#xmldiff-copy')!;
    const backBtn = container.querySelector<HTMLButtonElement>('#xmldiff-back')!;
    const forwardBtn = container.querySelector<HTMLButtonElement>('#xmldiff-forward')!;
    const statusEl = container.querySelector<HTMLElement>('#xmldiff-history-status')!;

    let lastResult = '';
    const cursor = new HistoryCursor('xmldiff');

    function updateNav() {
        backBtn.disabled = !cursor.canGoBack();
        forwardBtn.disabled = !cursor.canGoForward();
        statusEl.textContent = cursor.statusLabel();
    }

    cursor.refresh().then(() => {
        if (ctx.restore) cursor.anchorAt(ctx.restore.id);
        updateNav();
    });

    function loadResult(result: string) {
        summary.innerHTML = '';
        destroyDiffView(viewer);
        lastResult = '';
        if (result.trim() === 'No differences.') {
            summary.textContent = 'No differences.';
            return;
        }
        const counts = renderDiffView(viewer, result, 'xml');
        renderDiffSummary(summary, counts);
        lastResult = result;
        copyBtn.disabled = false;
    }

    container.querySelector('#xmldiff-run')!.addEventListener('click', async () => {
        error.textContent = '';
        copyBtn.disabled = true;
        try {
            loadResult(await RunDiffTool('xmldiff', left.getValue(), right.getValue()));
            cursor.resetToPresent();
            await cursor.refresh();
            updateNav();
        } catch (err) {
            error.textContent = String(err);
        }
    });

    // Loads a saved comparison and its stored diff (Back/Forward or opened
    // from History) without re-running it.
    function applySnapshot(snap: Snapshot) {
        left.setValue(snap.left ?? '');
        right.setValue(snap.right ?? '');
        copyBtn.disabled = true;
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

    copyBtn.addEventListener('click', async () => {
        try {
            await navigator.clipboard.writeText(lastResult);
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
        destroyDiffView(viewer);
        left.destroy();
        right.destroy();
    };
}

export const xmlDiffTool: ToolDef = { id: 'xmldiff', label: 'XML Diff', mount };
