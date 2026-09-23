import { RunTool } from '../../../wailsjs/go/main/App';
import type { ToolDef } from '../registry';

function mount(container: HTMLElement) {
    container.innerHTML = `
        <div class="tool-panel">
            <textarea id="jsontocsv-input" class="tool-input" placeholder="Paste JSON (array of objects, or a single object) here..."></textarea>
            <div class="tool-actions">
                <button id="jsontocsv-run" class="btn">Convert to CSV</button>
                <button id="jsontocsv-copy" class="btn btn-secondary" disabled>Copy</button>
            </div>
            <pre id="jsontocsv-output" class="tool-output"></pre>
            <p id="jsontocsv-error" class="tool-error"></p>
        </div>
    `;

    const input = container.querySelector<HTMLTextAreaElement>('#jsontocsv-input')!;
    const output = container.querySelector<HTMLElement>('#jsontocsv-output')!;
    const error = container.querySelector<HTMLElement>('#jsontocsv-error')!;
    const copyBtn = container.querySelector<HTMLButtonElement>('#jsontocsv-copy')!;

    container.querySelector('#jsontocsv-run')!.addEventListener('click', async () => {
        error.textContent = '';
        output.textContent = '';
        copyBtn.disabled = true;
        try {
            output.textContent = await RunTool('jsontocsv', input.value, {});
            copyBtn.disabled = false;
        } catch (err) {
            error.textContent = String(err);
        }
    });

    copyBtn.addEventListener('click', async () => {
        try {
            await navigator.clipboard.writeText(output.textContent ?? '');
            const original = copyBtn.textContent;
            copyBtn.textContent = 'Copied!';
            setTimeout(() => {
                copyBtn.textContent = original;
            }, 1200);
        } catch (err) {
            error.textContent = `Could not copy to clipboard: ${String(err)}`;
        }
    });
}

export const jsonToCsvTool: ToolDef = { id: 'jsontocsv', label: 'JSON to CSV', mount };
