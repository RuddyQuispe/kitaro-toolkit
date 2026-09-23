import { RunTool } from '../../../wailsjs/go/main/App';
import type { ToolDef } from '../registry';

function mount(container: HTMLElement) {
    container.innerHTML = `
        <div class="tool-panel">
            <textarea id="jsontocsv-input" class="tool-input" placeholder="Paste JSON (array of objects, or a single object) here..."></textarea>
            <div class="tool-actions">
                <button id="jsontocsv-run" class="btn">Convert to CSV</button>
            </div>
            <pre id="jsontocsv-output" class="tool-output"></pre>
            <p id="jsontocsv-error" class="tool-error"></p>
        </div>
    `;

    const input = container.querySelector<HTMLTextAreaElement>('#jsontocsv-input')!;
    const output = container.querySelector<HTMLElement>('#jsontocsv-output')!;
    const error = container.querySelector<HTMLElement>('#jsontocsv-error')!;

    container.querySelector('#jsontocsv-run')!.addEventListener('click', async () => {
        error.textContent = '';
        output.textContent = '';
        try {
            output.textContent = await RunTool('jsontocsv', input.value, {});
        } catch (err) {
            error.textContent = String(err);
        }
    });
}

export const jsonToCsvTool: ToolDef = { id: 'jsontocsv', label: 'JSON to CSV', mount };
