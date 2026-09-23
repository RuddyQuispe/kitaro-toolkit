import { RunTool } from '../../../wailsjs/go/main/App';
import type { ToolDef } from '../registry';

function mount(container: HTMLElement) {
    container.innerHTML = `
        <div class="tool-panel">
            <textarea id="jsonfmt-input" class="tool-input" placeholder="Paste JSON here..."></textarea>
            <div class="tool-actions">
                <select id="jsonfmt-mode">
                    <option value="pretty">Pretty</option>
                    <option value="minify">Minify</option>
                </select>
                <button id="jsonfmt-run" class="btn">Format</button>
            </div>
            <pre id="jsonfmt-output" class="tool-output"></pre>
            <p id="jsonfmt-error" class="tool-error"></p>
        </div>
    `;

    const input = container.querySelector<HTMLTextAreaElement>('#jsonfmt-input')!;
    const mode = container.querySelector<HTMLSelectElement>('#jsonfmt-mode')!;
    const output = container.querySelector<HTMLElement>('#jsonfmt-output')!;
    const error = container.querySelector<HTMLElement>('#jsonfmt-error')!;

    container.querySelector('#jsonfmt-run')!.addEventListener('click', async () => {
        error.textContent = '';
        output.textContent = '';
        try {
            output.textContent = await RunTool('jsonfmt', input.value, { mode: mode.value });
        } catch (err) {
            error.textContent = String(err);
        }
    });
}

export const jsonFormatterTool: ToolDef = { id: 'jsonfmt', label: 'JSON Formatter', mount };
