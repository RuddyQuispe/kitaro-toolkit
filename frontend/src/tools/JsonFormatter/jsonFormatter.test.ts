// @vitest-environment happy-dom
import { describe, expect, it, vi } from 'vitest';
import { EditorView } from 'codemirror';

const runTool = vi.fn();
vi.mock('../../../wailsjs/go/main/App', () => ({
    RunTool: (...args: unknown[]) => runTool(...args),
    ListHistory: async () => [{ id: '5', toolId: 'jsonfmt' }],
    GetHistoryEntry: vi.fn(),
}));

import { jsonFormatterTool } from './index';
import { history } from '../../../wailsjs/go/models';
import type { ToolContext } from '../registry';

function editorText(container: HTMLElement, selector: string): string {
    const dom = container.querySelector<HTMLElement>(`${selector} .cm-editor`)!;
    return EditorView.findFromDOM(dom)!.state.doc.toString();
}

describe('JSON Formatter restore', () => {
    it('loads input, output and mode from ctx.restore without re-running', () => {
        const container = document.createElement('div');
        const restore = history.Entry.createFrom({
            id: '5',
            toolId: 'jsonfmt',
            kind: 'run',
            input: '{ "a": 1 }',
            output: '{"a":1}',
            options: { mode: 'minify' },
        });
        const ctx: ToolContext = { navigate: vi.fn(), hasTool: () => true, restore };

        const cleanup = jsonFormatterTool.mount(container, ctx);

        expect(editorText(container, '#jsonfmt-input')).toBe('{ "a": 1 }');
        expect(editorText(container, '#jsonfmt-output')).toBe('{"a":1}');
        expect(container.querySelector<HTMLSelectElement>('#jsonfmt-mode')!.value).toBe('minify');
        expect(runTool).not.toHaveBeenCalled();
        cleanup?.();
    });

    it('ignores an unknown mode option', () => {
        const container = document.createElement('div');
        const restore = history.Entry.createFrom({ id: '5', toolId: 'jsonfmt', input: 'x', output: 'y', options: { mode: 'bogus' } });

        const cleanup = jsonFormatterTool.mount(container, { navigate: vi.fn(), hasTool: () => true, restore });

        expect(container.querySelector<HTMLSelectElement>('#jsonfmt-mode')!.value).toBe('pretty');
        cleanup?.();
    });
});
