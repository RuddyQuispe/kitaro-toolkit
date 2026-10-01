import { GetHistoryEntry, ListHistory } from '../../wailsjs/go/main/App';
import type { history } from '../../wailsjs/go/models';

// A Memento: an immutable snapshot of a tool's editable state, handed back
// unchanged on restore. history.Entry satisfies this shape (plus bookkeeping
// fields), so entries loaded from the backend double as snapshots directly.
export type Snapshot = history.Entry;

// The cursor itself only keeps ListHistory's lightweight summaries (ids,
// previews); back()/forward() fetch the full entry they land on.
//
// Per-tool undo/redo cursor (the Caretaker) over that tool's own entries in
// the shared history log (newest first). position -1 means "nothing
// browsed yet" (a fresh mount with blank fields); 0 means "anchored at the
// most recent entry" (typically right after a run, since that entry's
// content is already what's on screen); higher positions go further back.
// Both -1 and 0 render as "Present" and back() treats them the same way —
// the distinction only matters so the very first Back click after a run
// skips straight to the *previous* requirement instead of re-showing the
// one already on screen.
export class HistoryCursor {
    private entries: history.Entry[] = [];
    private position = -1;

    constructor(private toolId: string) {}

    async refresh(): Promise<void> {
        const all = await ListHistory();
        this.entries = (all ?? []).filter((e) => e.toolId === this.toolId);
        if (this.position >= this.entries.length) {
            this.position = -1;
        }
    }

    // Call after a fresh run: its result is now the most recent entry, and
    // is already what's on screen.
    resetToPresent(): void {
        this.position = 0;
    }

    // Call after loading an entry from outside the cursor (e.g. opened from
    // the History screen) so back/forward continue from it. Unknown ids
    // fall back to "Present".
    anchorAt(entryId: string): void {
        this.position = this.entries.findIndex((e) => e.id === entryId);
    }

    canGoBack(): boolean {
        return this.position + 1 < this.entries.length;
    }

    canGoForward(): boolean {
        return this.position > 0;
    }

    back(): Promise<Snapshot | null> {
        if (!this.canGoBack()) return Promise.resolve(null);
        return this.moveTo(this.position + 1);
    }

    forward(): Promise<Snapshot | null> {
        if (!this.canGoForward()) return Promise.resolve(null);
        return this.moveTo(this.position - 1);
    }

    // Only moves once the fetch succeeds, so a failed load leaves the
    // cursor where the screen still is.
    private async moveTo(target: number): Promise<Snapshot> {
        const entry = await GetHistoryEntry(this.entries[target].id);
        this.position = target;
        return entry;
    }

    statusLabel(): string {
        if (this.position <= 0) return 'Present';
        return `${this.position} of ${this.entries.length - 1} back`;
    }
}

// Restores a saved option into a <select>, ignoring values it no longer
// offers.
export function restoreSelect(select: HTMLSelectElement, value: string | undefined): void {
    if (value && Array.from(select.options).some((o) => o.value === value)) {
        select.value = value;
    }
}
