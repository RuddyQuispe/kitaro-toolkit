import { RunDiffTool } from '../../../wailsjs/go/main/App';
import type { ToolContext, ToolDef } from '../registry';
import { createCodeEditor } from '../../components/codeEditor';
import { destroyDiffView, renderDiffView } from '../../components/diffViewer';
import { renderDiffSummary } from '../../components/diffSummary';
import { HistoryCursor, type Snapshot } from '../historyCursor';
import { DIALECTS } from '../SqlFormatter';
import { normalizeSql } from './normalize';
import type { SqlLanguage } from 'sql-formatter';

function mount(container: HTMLElement, ctx: ToolContext) {
    const options = Object.keys(DIALECTS)
        .map((label) => `<option value="${DIALECTS[label]}">${label}</option>`)
        .join('');

    container.innerHTML = `
        <div class="tool-panel diff-panel">
            <div class="diff-inputs">
                <div id="sqldiff-left" class="tool-input"></div>
                <div id="sqldiff-right" class="tool-input"></div>
            </div>
            <div class="tool-actions">
                <select id="sqldiff-dialect">${options}</select>
                <button id="sqldiff-run" class="btn">Compare</button>
                <button id="sqldiff-back" class="btn btn-secondary history-nav-btn" title="Previous requirement">◀</button>
                <button id="sqldiff-forward" class="btn btn-secondary history-nav-btn" title="Next requirement">▶</button>
                <button id="sqldiff-copy" class="btn btn-secondary" disabled>Copy</button>
                <span id="sqldiff-history-status" class="history-nav-status tool-hint"></span>
            </div>
            <p id="sqldiff-summary" class="diff-summary"></p>
            <div id="sqldiff-viewer" class="diff-viewer"></div>
            <p id="sqldiff-error" class="tool-error"></p>
        </div>
    `;

    const left = createCodeEditor(container.querySelector<HTMLElement>('#sqldiff-left')!, { lang: 'sql', placeholder: 'Left SQL...' });
    const right = createCodeEditor(container.querySelector<HTMLElement>('#sqldiff-right')!, { lang: 'sql', placeholder: 'Right SQL...' });
    const dialect = container.querySelector<HTMLSelectElement>('#sqldiff-dialect')!;
    const summary = container.querySelector<HTMLElement>('#sqldiff-summary')!;
    const viewer = container.querySelector<HTMLElement>('#sqldiff-viewer')!;
    const error = container.querySelector<HTMLElement>('#sqldiff-error')!;
    const copyBtn = container.querySelector<HTMLButtonElement>('#sqldiff-copy')!;
    const backBtn = container.querySelector<HTMLButtonElement>('#sqldiff-back')!;
    const forwardBtn = container.querySelector<HTMLButtonElement>('#sqldiff-forward')!;
    const statusEl = container.querySelector<HTMLElement>('#sqldiff-history-status')!;

    let lastResult = '';
    const cursor = new HistoryCursor('sqldiff');

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
        const counts = renderDiffView(viewer, result, 'sql');
        renderDiffSummary(summary, counts);
        lastResult = result;
        copyBtn.disabled = false;
    }

    container.querySelector('#sqldiff-run')!.addEventListener('click', async () => {
        error.textContent = '';
        copyBtn.disabled = true;
        try {
            // Normalize both sides with sql-formatter first so layout and
            // keyword casing never show up as differences.
            const language = dialect.value as SqlLanguage;
            let normLeft: string;
            let normRight: string;
            try {
                normLeft = normalizeSql(left.getValue(), language);
            } catch (err) {
                throw new Error(`invalid left SQL: ${String(err)}`);
            }
            try {
                normRight = normalizeSql(right.getValue(), language);
            } catch (err) {
                throw new Error(`invalid right SQL: ${String(err)}`);
            }
            loadResult(await RunDiffTool('sqldiff', normLeft, normRight));
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

export const sqlDiffTool: ToolDef = { id: 'sqldiff', label: 'SQL Diff', mount };
