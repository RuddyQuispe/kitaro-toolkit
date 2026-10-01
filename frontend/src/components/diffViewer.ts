import { EditorView, basicSetup } from 'codemirror';
import { EditorState, StateEffect, StateField } from '@codemirror/state';
import { Decoration, type DecorationSet } from '@codemirror/view';
import { showMinimap } from '@replit/codemirror-minimap';
import { editorTheme, langExtension } from './codeEditor';

export type DiffLang = 'json' | 'xml' | 'sql';

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
// Same palette as the .count-add/.count-del/.count-chg summary text and the
// (translucent) .cm-diff-add/del/chg line backgrounds in shell.css, but at
// full opacity since minimap gutter marks are only a couple pixels tall.
const MINIMAP_COLORS = { add: '#2f9e44', remove: '#e03131', change: '#f08c00' } as const;

// One live EditorView per container. Clearing a container with innerHTML
// only detaches the view's DOM; its listeners, observers and the minimap
// keep the whole editor alive, so every view must be destroy()ed.
const views = new WeakMap<HTMLElement, EditorView>();

/** Destroys the diff view rendered into `container` (if any) and empties it. */
export function destroyDiffView(container: HTMLElement): void {
    views.get(container)?.destroy();
    views.delete(container);
    container.innerHTML = '';
}

export function renderDiffView(container: HTMLElement, raw: string, lang: DiffLang): DiffSummary {
    destroyDiffView(container);

    const lines = parseDiffLines(raw);
    const summary: DiffSummary = { added: 0, removed: 0, changed: 0 };

    const minimapGutters: Record<number, string> = {};
    lines.forEach((line, i) => {
        if (line.marker === '+') minimapGutters[i + 1] = MINIMAP_COLORS.add;
        else if (line.marker === '-') minimapGutters[i + 1] = MINIMAP_COLORS.remove;
        else if (line.marker === '~') minimapGutters[i + 1] = MINIMAP_COLORS.change;
    });

    const view = new EditorView({
        state: EditorState.create({
            doc: lines.map((l) => l.text).join('\n'),
            extensions: [
                basicSetup,
                langExtension(lang),
                EditorView.editable.of(false),
                diffDecorationsField,
                showMinimap.compute(['doc'], () => ({
                    create: () => ({ dom: document.createElement('div') }),
                    displayText: 'blocks',
                    showOverlay: 'always',
                    gutters: [minimapGutters],
                    eventHandlers: {
                        // Click anywhere on the minimap to jump straight to that
                        // line in the main editor — the point of the minimap is
                        // fast navigation to a difference, not just an overview.
                        click: (e, v) => {
                            const target = e.currentTarget as HTMLElement;
                            const rect = target.getBoundingClientRect();
                            const ratio = (e.clientY - rect.top) / rect.height;
                            const lineNumber = Math.max(
                                1,
                                Math.min(v.state.doc.lines, Math.round(ratio * v.state.doc.lines)),
                            );
                            const pos = v.state.doc.line(lineNumber).from;
                            v.dispatch({ effects: EditorView.scrollIntoView(pos, { y: 'center' }) });
                        },
                    },
                })),
                editorTheme,
            ],
        }),
        parent: container,
    });
    views.set(container, view);

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
