import { RunDiffTool } from '../../../wailsjs/go/main/App';
import type { ToolDef } from '../registry';
import { renderDiffView } from '../../components/diffViewer';
import { renderDiffSummary } from '../../components/diffSummary';

function mount(container: HTMLElement) {
    container.innerHTML = `
        <div class="tool-panel diff-panel">
            <div class="diff-inputs">
                <textarea id="jsondiff-left" class="tool-input" placeholder="Left JSON..."></textarea>
                <textarea id="jsondiff-right" class="tool-input" placeholder="Right JSON..."></textarea>
            </div>
            <div class="tool-actions">
                <button id="jsondiff-run" class="btn">Compare</button>
            </div>
            <p id="jsondiff-summary" class="diff-summary"></p>
            <div id="jsondiff-viewer" class="diff-viewer"></div>
            <p id="jsondiff-error" class="tool-error"></p>
        </div>
    `;

    const left = container.querySelector<HTMLTextAreaElement>('#jsondiff-left')!;
    const right = container.querySelector<HTMLTextAreaElement>('#jsondiff-right')!;
    const summary = container.querySelector<HTMLElement>('#jsondiff-summary')!;
    const viewer = container.querySelector<HTMLElement>('#jsondiff-viewer')!;
    const error = container.querySelector<HTMLElement>('#jsondiff-error')!;

    container.querySelector('#jsondiff-run')!.addEventListener('click', async () => {
        error.textContent = '';
        summary.innerHTML = '';
        viewer.innerHTML = '';
        try {
            const result = await RunDiffTool('jsondiff', left.value, right.value);
            if (result.trim() === 'No differences.') {
                summary.textContent = 'No differences.';
                return;
            }
            const counts = renderDiffView(viewer, result, 'json');
            renderDiffSummary(summary, counts);
        } catch (err) {
            error.textContent = String(err);
        }
    });
}

export const jsonDiffTool: ToolDef = { id: 'jsondiff', label: 'JSON Diff', mount };
