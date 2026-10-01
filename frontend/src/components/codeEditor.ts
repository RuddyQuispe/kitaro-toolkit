import { EditorView, basicSetup } from 'codemirror';
import { EditorState, type Extension } from '@codemirror/state';
import { placeholder as placeholderExt } from '@codemirror/view';
import { json } from '@codemirror/lang-json';
import { xml } from '@codemirror/lang-xml';
import { sql } from '@codemirror/lang-sql';

export type EditorLang = 'json' | 'xml' | 'sql' | 'text';

export interface CodeEditorOptions {
    lang?: EditorLang;
    readOnly?: boolean;
    placeholder?: string;
}

export interface CodeEditor {
    getValue(): string;
    setValue(text: string): void;
    destroy(): void;
    view: EditorView;
}

/** Shared look for every CodeMirror view (tool inputs/outputs, diff viewer). */
export const editorTheme = EditorView.theme({
    '&': { height: '100%', fontSize: 'var(--editor-font-size)', backgroundColor: 'transparent', color: 'var(--fg)' },
    '.cm-scroller': { overflow: 'auto', fontFamily: 'ui-monospace, SFMono-Regular, Menlo, Consolas, monospace' },
    '.cm-gutters': { backgroundColor: 'transparent', color: 'var(--fg)', opacity: 0.5, border: 'none' },
    '.cm-activeLine': { backgroundColor: 'transparent' },
    '.cm-activeLineGutter': { backgroundColor: 'transparent' },
});

export function langExtension(lang: EditorLang): Extension {
    if (lang === 'json') return json();
    if (lang === 'xml') return xml();
    if (lang === 'sql') return sql();
    return [];
}

/**
 * Mounts a CodeMirror editor (line numbers, folding, search) into `host`.
 * CodeMirror only renders the visible lines, so multi-MB pastes stay
 * responsive. Callers must destroy() it when the tool unmounts.
 */
export function createCodeEditor(host: HTMLElement, options: CodeEditorOptions = {}): CodeEditor {
    const { lang = 'text', readOnly = false, placeholder } = options;

    const extensions: Extension[] = [basicSetup, langExtension(lang), editorTheme];
    if (placeholder) extensions.push(placeholderExt(placeholder));
    if (readOnly) {
        // Still selectable and copyable, just not editable by the user.
        extensions.push(EditorState.readOnly.of(true), EditorView.editable.of(false));
    }

    const view = new EditorView({
        state: EditorState.create({ doc: '', extensions }),
        parent: host,
    });

    return {
        view,
        getValue: () => view.state.doc.toString(),
        setValue(text: string) {
            view.dispatch({ changes: { from: 0, to: view.state.doc.length, insert: text } });
        },
        destroy: () => view.destroy(),
    };
}
