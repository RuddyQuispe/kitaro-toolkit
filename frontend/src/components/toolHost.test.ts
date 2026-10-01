// @vitest-environment happy-dom
import { describe, expect, it, vi } from 'vitest';
import { createToolHost } from './toolHost';
import type { ToolContext } from '../tools/registry';
import { history } from '../../wailsjs/go/models';

function makeCtx(restore?: history.Entry): ToolContext {
    return { navigate: vi.fn(), hasTool: () => true, restore };
}

describe('createToolHost', () => {
    it('runs the previous tool cleanup before mounting the next tool', () => {
        const content = document.createElement('div');
        const host = createToolHost(content);
        const cleanupA = vi.fn();
        const mountB = vi.fn((el: HTMLElement) => {
            expect(cleanupA).toHaveBeenCalledTimes(1);
            el.textContent = 'B';
        });

        host.show({ id: 'a', label: 'A', mount: (el) => { el.textContent = 'A'; return cleanupA; } }, makeCtx());
        host.show({ id: 'b', label: 'B', mount: mountB }, makeCtx());

        expect(cleanupA).toHaveBeenCalledTimes(1);
        expect(content.textContent).toBe('B');
    });

    it('accepts tools whose mount returns no cleanup', () => {
        const content = document.createElement('div');
        const host = createToolHost(content);
        host.show({ id: 'a', label: 'A', mount: () => {} }, makeCtx());
        expect(() => host.show({ id: 'b', label: 'B', mount: () => {} }, makeCtx())).not.toThrow();
    });

    it('passes the context, including the entry to restore, to mount', () => {
        const content = document.createElement('div');
        const host = createToolHost(content);
        const entry = history.Entry.createFrom({ id: '7', toolId: 'a', input: 'x', output: 'y' });
        const ctx = makeCtx(entry);
        const mount = vi.fn();

        host.show({ id: 'a', label: 'A', mount }, ctx);

        expect(mount).toHaveBeenCalledWith(content, ctx);
        expect(mount.mock.calls[0][1].restore).toBe(entry);
    });
});
