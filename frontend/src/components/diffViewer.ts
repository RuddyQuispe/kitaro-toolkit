import { EditorView, basicSetup } from 'codemirror';
import { EditorState, StateEffect, StateField } from '@codemirror/state';
import { Decoration, type DecorationSet } from '@codemirror/view';
import { json } from '@codemirror/lang-json';
import { xml } from '@codemirror/lang-xml';

export type DiffLang = 'json' | 'xml';

interface DiffLine {
    marker: '+' | '-' | '~' | ' ';
    text: string;
}

// Backend diff output prefixes each line with a one-character marker
// ("+ " added, "- " removed, "~ " changed, " " unchanged/context) — see
// internal/tools/xmldiff and gojsondiff's AsciiFormatter.
function parseDiffLines(raw: string): DiffLine[] {
    return raw.split('\n').map((line) => {
        const marker = line[0];
        if (marker === '+' || marker === '-' || marker === '~') {
            return { marker, text: line.slice(1) };
        }
        return { marker: ' ', text: line.slice(1) || line };
    });
}

const diffLineMark = Decoration.line({ attributes: { class: 'cm-diff-line' } });
const addLineMark = Decoration.line({ attributes: { class: 'cm-diff-line cm-diff-add' } });
const delLineMark = Decoration.line({ attributes: { class: 'cm-diff-line cm-diff-del' } });
const chgLineMark = Decoration.line({ attributes: { class: 'cm-diff-line cm-diff-chg' } });

const setDiffDecorations = StateEffect.define<DecorationSet>();

const diffDecorationsField = StateField.define<DecorationSet>({
    create: () => Decoration.none,
    update(value, tr) {
        for (const effect of tr.effects) {
            if (effect.is(setDiffDecorations)) return effect.value;
        }
        return value.map(tr.changes);
    },
    provide: (f) => EditorView.decorations.from(f),
});

function langExtension(lang: DiffLang) {
    return lang === 'xml' ? xml() : json();
}

/** Counts of each diff marker across the parsed lines. */
export interface DiffSummary {
    added: number;
    removed: number;
    changed: number;
}

/**
 * Renders diff text (marker-prefixed lines) into a read-only, syntax
 * highlighted CodeMirror view inside `container`, with per-line
 * highlighting for added/removed/changed lines. Returns the counts so the
 * caller can render a "N differences found" summary.
 */
export function renderDiffView(container: HTMLElement, raw: string, lang: DiffLang): DiffSummary {
    container.innerHTML = '';

    const lines = parseDiffLines(raw);
    const summary: DiffSummary = { added: 0, removed: 0, changed: 0 };

    const view = new EditorView({
        state: EditorState.create({
            doc: lines.map((l) => l.text).join('\n'),
            extensions: [
                basicSetup,
                langExtension(lang),
                EditorView.editable.of(false),
                diffDecorationsField,
                EditorView.theme({
                    '&': { height: '100%', fontSize: '0.85rem', backgroundColor: 'transparent', color: 'var(--fg)' },
                    '.cm-scroller': { overflow: 'auto', fontFamily: 'ui-monospace, SFMono-Regular, Menlo, Consolas, monospace' },
                    '.cm-gutters': { backgroundColor: 'transparent', color: 'var(--fg)', opacity: 0.5, border: 'none' },
                    '.cm-activeLine': { backgroundColor: 'transparent' },
                    '.cm-activeLineGutter': { backgroundColor: 'transparent' },
                }),
            ],
        }),
        parent: container,
    });

    const builder: { from: number; to: number; mark: typeof diffLineMark }[] = [];
    let pos = 0;
    for (const line of lines) {
        const from = pos;
        if (line.marker === '+') {
            summary.added++;
            builder.push({ from, to: from, mark: addLineMark });
        } else if (line.marker === '-') {
            summary.removed++;
            builder.push({ from, to: from, mark: delLineMark });
        } else if (line.marker === '~') {
            summary.changed++;
            builder.push({ from, to: from, mark: chgLineMark });
        }
        pos += line.text.length + 1; // +1 for the newline joining lines
    }

    const decorations = Decoration.set(
        builder.map((b) => b.mark.range(b.from)),
        true,
    );
    view.dispatch({ effects: setDiffDecorations.of(decorations) });

    return summary;
}
