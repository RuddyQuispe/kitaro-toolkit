import { RunTool } from '../../../wailsjs/go/main/App';
import type { ToolDef } from '../registry';

function mount(container: HTMLElement) {
    container.innerHTML = `
        <div class="tool-panel">
            <textarea id="xmlfmt-input" class="tool-input" placeholder="Paste XML here..."></textarea>
            <div class="tool-actions">
                <select id="xmlfmt-mode">
                    <option value="pretty">Pretty</option>
                    <option value="minify">Minify</option>
                </select>
                <button id="xmlfmt-run" class="btn">Format</button>
            </div>
            <pre id="xmlfmt-output" class="tool-output"></pre>
            <p id="xmlfmt-error" class="tool-error"></p>
        </div>
    `;

    const input = container.querySelector<HTMLTextAreaElement>('#xmlfmt-input')!;
    const mode = container.querySelector<HTMLSelectElement>('#xmlfmt-mode')!;
    const output = container.querySelector<HTMLElement>('#xmlfmt-output')!;
    const error = container.querySelector<HTMLElement>('#xmlfmt-error')!;

    container.querySelector('#xmlfmt-run')!.addEventListener('click', async () => {
        error.textContent = '';
        output.textContent = '';
        try {
            output.textContent = await RunTool('xmlfmt', input.value, { mode: mode.value });
        } catch (err) {
            error.textContent = String(err);
        }
    });
}

export const xmlFormatterTool: ToolDef = { id: 'xmlfmt', label: 'XML Formatter', mount };
