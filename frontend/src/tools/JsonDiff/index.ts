import { RunDiffTool } from '../../../wailsjs/go/main/App';
import type { ToolDef } from '../registry';

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
            <pre id="jsondiff-output" class="tool-output"></pre>
            <p id="jsondiff-error" class="tool-error"></p>
        </div>
    `;

    const left = container.querySelector<HTMLTextAreaElement>('#jsondiff-left')!;
    const right = container.querySelector<HTMLTextAreaElement>('#jsondiff-right')!;
    const output = container.querySelector<HTMLElement>('#jsondiff-output')!;
    const error = container.querySelector<HTMLElement>('#jsondiff-error')!;

    container.querySelector('#jsondiff-run')!.addEventListener('click', async () => {
        error.textContent = '';
        output.textContent = '';
        try {
            output.textContent = await RunDiffTool('jsondiff', left.value, right.value);
        } catch (err) {
            error.textContent = String(err);
        }
    });
}

export const jsonDiffTool: ToolDef = { id: 'jsondiff', label: 'JSON Diff', mount };
