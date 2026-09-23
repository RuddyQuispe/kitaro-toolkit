import { RunDiffTool } from '../../../wailsjs/go/main/App';
import type { ToolDef } from '../registry';

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
            <pre id="xmldiff-output" class="tool-output"></pre>
            <p id="xmldiff-error" class="tool-error"></p>
        </div>
    `;

    const left = container.querySelector<HTMLTextAreaElement>('#xmldiff-left')!;
    const right = container.querySelector<HTMLTextAreaElement>('#xmldiff-right')!;
    const output = container.querySelector<HTMLElement>('#xmldiff-output')!;
    const error = container.querySelector<HTMLElement>('#xmldiff-error')!;

    container.querySelector('#xmldiff-run')!.addEventListener('click', async () => {
        error.textContent = '';
        output.textContent = '';
        try {
            output.textContent = await RunDiffTool('xmldiff', left.value, right.value);
        } catch (err) {
            error.textContent = String(err);
        }
    });
}

export const xmlDiffTool: ToolDef = { id: 'xmldiff', label: 'XML Diff', mount };
