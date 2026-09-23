import type { DiffSummary } from './diffViewer';

/** Renders a "N differences found (+a -r ~c)" line into `el`. */
export function renderDiffSummary(el: HTMLElement, s: DiffSummary) {
    const total = s.added + s.removed + s.changed;
    if (total === 0) {
        el.textContent = 'No differences.';
        return;
    }
    el.innerHTML = `${total} difference${total === 1 ? '' : 's'} found `
        + `(<span class="count-add">+${s.added}</span> `
        + `<span class="count-del">-${s.removed}</span> `
        + `<span class="count-chg">~${s.changed}</span>)`;
}
