// @vitest-environment happy-dom
import { afterEach, describe, expect, it, vi } from 'vitest';
import { EditorView } from 'codemirror';
import { createCodeEditor } from './codeEditor';

describe('createCodeEditor', () => {
    afterEach(() => vi.restoreAllMocks());

    it('round-trips text through setValue/getValue', () => {
        const host = document.createElement('div');
        const editor = createCodeEditor(host, { lang: 'json' });

        expect(editor.getValue()).toBe('');
        editor.setValue('{\n  "a": 1\n}');
        expect(editor.getValue()).toBe('{\n  "a": 1\n}');
        editor.setValue('');
        expect(editor.getValue()).toBe('');

        editor.destroy();
    });

    it('renders a line-number gutter', () => {
        const host = document.createElement('div');
        const editor = createCodeEditor(host, { lang: 'text' });

        expect(host.querySelector('.cm-lineNumbers')).not.toBeNull();

        editor.destroy();
    });

    it('blocks user edits when readOnly, but setValue still works', () => {
        const host = document.createElement('div');
        const editor = createCodeEditor(host, { lang: 'sql', readOnly: true });

        expect(editor.view.state.readOnly).toBe(true);
        expect(editor.view.contentDOM.getAttribute('contenteditable')).toBe('false');
        editor.setValue('SELECT 1');
        expect(editor.getValue()).toBe('SELECT 1');

        editor.destroy();
    });

    it('is editable by default', () => {
        const host = document.createElement('div');
        const editor = createCodeEditor(host);

        expect(editor.view.state.readOnly).toBe(false);
        expect(editor.view.contentDOM.getAttribute('contenteditable')).toBe('true');

        editor.destroy();
    });

    it('destroy() destroys the view and removes its DOM', () => {
        const destroy = vi.spyOn(EditorView.prototype, 'destroy');
        const host = document.createElement('div');
        const editor = createCodeEditor(host, { lang: 'xml', placeholder: 'Paste XML...' });

        expect(host.querySelectorAll('.cm-editor')).toHaveLength(1);
        editor.destroy();

        expect(destroy).toHaveBeenCalledTimes(1);
        expect(host.querySelector('.cm-editor')).toBeNull();
    });
});
