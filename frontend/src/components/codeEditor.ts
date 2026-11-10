import { EditorView, basicSetup } from 'codemirror';
import { EditorState, type Extension } from '@codemirror/state';
import { placeholder as placeholderExt } from '@codemirror/view';
import { HighlightStyle, syntaxHighlighting } from '@codemirror/language';
import { tags as t } from '@lezer/highlight';
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

const editorBaseTheme = EditorView.theme({
    '&': { height: '100%', fontSize: 'var(--editor-font-size)', backgroundColor: 'transparent', color: 'var(--editor-text)' },
    '.cm-scroller': { overflow: 'auto', fontFamily: 'ui-monospace, SFMono-Regular, Menlo, Consolas, monospace' },
    '.cm-gutters': { backgroundColor: 'transparent', color: 'var(--fg)', opacity: 0.5, border: 'none' },
    '.cm-activeLine': { backgroundColor: 'transparent' },
    '.cm-activeLineGutter': { backgroundColor: 'transparent' },
    // CodeMirror's base theme assumes a light background (black cursor,
    // light-gray selection), which is unreadable in dark mode.
    '.cm-content': { caretColor: 'var(--editor-cursor)' },
    '.cm-cursor, .cm-dropCursor': { borderLeftColor: 'var(--editor-cursor)' },
    '.cm-selectionBackground, &.cm-focused > .cm-scroller > .cm-selectionLayer .cm-selectionBackground, .cm-content ::selection':
        { backgroundColor: 'var(--editor-selection)' },
    '.cm-panels': { backgroundColor: 'var(--editor-panel-bg)', color: 'var(--editor-text)' },
    '.cm-tooltip': { backgroundColor: 'var(--editor-panel-bg)', color: 'var(--editor-text)', border: '1px solid var(--border)' },
    '.cm-placeholder': { color: 'var(--tok-comment)' },
});

/**
 * Tokyo Night syntax colors. Every color is a CSS variable defined per
 * data-theme in theme/theme.css (Tokyo Night / Tokyo Night Day), so toggling
 * the theme recolors open editors without reconfiguring them. It replaces
 * basicSetup's light-only default style, which is registered as a fallback.
 */
const tokyoNightHighlight = HighlightStyle.define([
    { tag: [t.keyword, t.operatorKeyword, t.modifier, t.controlKeyword], color: 'var(--tok-keyword)' },
    { tag: [t.string, t.special(t.string), t.regexp], color: 'var(--tok-string)' },
    { tag: [t.number, t.integer, t.float], color: 'var(--tok-number)' },
    { tag: [t.bool, t.null, t.atom], color: 'var(--tok-atom)' },
    { tag: t.propertyName, color: 'var(--tok-property)' },
    { tag: [t.function(t.variableName), t.function(t.propertyName), t.standard(t.name)], color: 'var(--tok-function)' },
    { tag: [t.typeName, t.className, t.namespace], color: 'var(--tok-type)' },
    { tag: [t.variableName, t.name, t.special(t.name)], color: 'var(--tok-variable)' },
    { tag: [t.operator, t.compareOperator, t.arithmeticOperator, t.logicOperator], color: 'var(--tok-operator)' },
    { tag: [t.punctuation, t.separator, t.bracket, t.paren, t.squareBracket, t.brace], color: 'var(--tok-punctuation)' },
    { tag: [t.comment, t.lineComment, t.blockComment], color: 'var(--tok-comment)', fontStyle: 'italic' },
    { tag: [t.tagName, t.angleBracket], color: 'var(--tok-tag)' },
    { tag: t.attributeName, color: 'var(--tok-attribute)' },
    { tag: [t.processingInstruction, t.documentMeta, t.meta], color: 'var(--tok-comment)' },
    { tag: t.invalid, color: 'var(--tok-invalid)' },
]);

/** Shared look for every CodeMirror view (tool inputs/outputs, diff viewer). */
export const editorTheme: Extension = [editorBaseTheme, syntaxHighlighting(tokyoNightHighlight)];

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
