import { beforeEach, describe, expect, it, vi } from 'vitest';

const listHistory = vi.fn();
const getHistoryEntry = vi.fn();
vi.mock('../../wailsjs/go/main/App', () => ({
    ListHistory: () => listHistory(),
    GetHistoryEntry: (id: string) => getHistoryEntry(id),
}));

import { HistoryCursor } from './historyCursor';

const summaries = [
    { id: '3', toolId: 'jsonfmt' },
    { id: '2', toolId: 'xmlfmt' },
    { id: '1', toolId: 'jsonfmt' },
];

describe('HistoryCursor', () => {
    beforeEach(() => {
        listHistory.mockReset().mockResolvedValue(summaries);
        getHistoryEntry.mockReset().mockImplementation(async (id: string) => ({ id, toolId: 'jsonfmt', input: `in-${id}`, output: `out-${id}` }));
    });

    it('fetches the full entry when moving back and forward', async () => {
        const cursor = new HistoryCursor('jsonfmt');
        await cursor.refresh();
        cursor.resetToPresent();

        const back = await cursor.back();
        expect(getHistoryEntry).toHaveBeenLastCalledWith('1');
        expect(back?.input).toBe('in-1');
        expect(cursor.statusLabel()).toBe('1 of 1 back');

        const forward = await cursor.forward();
        expect(getHistoryEntry).toHaveBeenLastCalledWith('3');
        expect(forward?.output).toBe('out-3');
        expect(cursor.statusLabel()).toBe('Present');
    });

    it('does not move when the fetch fails', async () => {
        const cursor = new HistoryCursor('jsonfmt');
        await cursor.refresh();
        cursor.resetToPresent();
        getHistoryEntry.mockRejectedValueOnce(new Error('boom'));

        await expect(cursor.back()).rejects.toThrow('boom');
        expect(cursor.statusLabel()).toBe('Present');
        expect(cursor.canGoBack()).toBe(true);
    });

    it('anchors at a restored entry so back/forward continue from it', async () => {
        const cursor = new HistoryCursor('jsonfmt');
        await cursor.refresh();

        cursor.anchorAt('1');
        expect(cursor.statusLabel()).toBe('1 of 1 back');
        expect(cursor.canGoBack()).toBe(false);
        expect(cursor.canGoForward()).toBe(true);

        cursor.anchorAt('missing');
        expect(cursor.statusLabel()).toBe('Present');
    });
});
