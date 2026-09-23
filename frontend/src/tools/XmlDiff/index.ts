import { RunDiffTool } from '../../../wailsjs/go/main/App';
import type { ToolDef } from '../registry';
import { renderDiffView } from '../../components/diffViewer';
import { renderDiffSummary } from '../../components/diffSummary';

function mount(container: HTMLElement) {
    container.innerHTML = `
        <div class="tool-panel diff-panel">
            <div class="diff-inputs">
                <textarea id="xmldiff-left" class="tool-input" placeholder="Left XML..."></textarea>
                <textarea id="xmldiff-right" class="tool-input" placeholder="Right XML..."></textarea>
            </div>
            <div class="tool-actions">
                <button id="xmldiff-run" class="btn">Compare</button>
            </div>
            <p id="xmldiff-summary" class="diff-summary"></p>
            <div id="xmldiff-viewer" class="diff-viewer"></div>
            <p id="xmldiff-error" class="tool-error"></p>
        </div>
    `;

    const left = container.querySelector<HTMLTextAreaElement>('#xmldiff-left')!;
    const right = container.querySelector<HTMLTextAreaElement>('#xmldiff-right')!;
    const summary = container.querySelector<HTMLElement>('#xmldiff-summary')!;
    const viewer = container.querySelector<HTMLElement>('#xmldiff-viewer')!;
    const error = container.querySelector<HTMLElement>('#xmldiff-error')!;

    container.querySelector('#xmldiff-run')!.addEventListener('click', async () => {
        error.textContent = '';
        summary.innerHTML = '';
        viewer.innerHTML = '';
        try {
            const result = await RunDiffTool('xmldiff', left.value, right.value);
            if (result.trim() === 'No differences.') {
                summary.textContent = 'No differences.';
                return;
            }
            const counts = renderDiffView(viewer, result, 'xml');
            renderDiffSummary(summary, counts);
        } catch (err) {
            error.textContent = String(err);
        }
    });
}

export const xmlDiffTool: ToolDef = { id: 'xmldiff', label: 'XML Diff', mount };
