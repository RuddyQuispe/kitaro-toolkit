// @vitest-environment happy-dom
import { afterEach, describe, expect, it, vi } from 'vitest';
import { EditorView } from 'codemirror';
import { destroyDiffView, renderDiffView } from './diffViewer';

const DIFF = ' {\n+  "a": 1\n- "b": 2\n }';

describe('renderDiffView lifecycle', () => {
    afterEach(() => vi.restoreAllMocks());

    it('destroys the previous view when re-rendering into the same container', () => {
        const destroy = vi.spyOn(EditorView.prototype, 'destroy');
        const container = document.createElement('div');

        renderDiffView(container, DIFF, 'json');
        renderDiffView(container, DIFF, 'json');

        expect(destroy).toHaveBeenCalledTimes(1);
        expect(container.querySelectorAll('.cm-editor')).toHaveLength(1);
    });

    it('destroyDiffView destroys the current view and empties the container', () => {
        const destroy = vi.spyOn(EditorView.prototype, 'destroy');
        const container = document.createElement('div');

        renderDiffView(container, DIFF, 'json');
        destroyDiffView(container);
        destroyDiffView(container); // idempotent

        expect(destroy).toHaveBeenCalledTimes(1);
        expect(container.childElementCount).toBe(0);
    });
});
