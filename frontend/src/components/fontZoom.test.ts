import { describe, expect, it } from 'vitest';
import {
    DEFAULT_FONT_SIZE,
    MAX_FONT_SIZE,
    MIN_FONT_SIZE,
    clampFontSize,
    nextFontSize,
} from './fontZoom';

describe('font size bounds', () => {
    it('defaults to 14px within a 10..32px range', () => {
        expect(DEFAULT_FONT_SIZE).toBe(14);
        expect(MIN_FONT_SIZE).toBe(10);
        expect(MAX_FONT_SIZE).toBe(32);
    });
});

describe('nextFontSize', () => {
    it('grows by 1px when scrolling up (negative deltaY)', () => {
        expect(nextFontSize(14, -100)).toBe(15);
    });

    it('shrinks by 1px when scrolling down (positive deltaY)', () => {
        expect(nextFontSize(14, 100)).toBe(13);
    });

    it('keeps the size when deltaY is 0', () => {
        expect(nextFontSize(14, 0)).toBe(14);
    });

    it('steps by 1px regardless of delta magnitude', () => {
        expect(nextFontSize(14, -3)).toBe(15);
        expect(nextFontSize(14, 1200)).toBe(13);
    });

    it('never goes above the max', () => {
        expect(nextFontSize(MAX_FONT_SIZE, -100)).toBe(MAX_FONT_SIZE);
    });

    it('never goes below the min', () => {
        expect(nextFontSize(MIN_FONT_SIZE, 100)).toBe(MIN_FONT_SIZE);
    });
});

describe('clampFontSize', () => {
    it('bounds out-of-range values', () => {
        expect(clampFontSize(2)).toBe(MIN_FONT_SIZE);
        expect(clampFontSize(99)).toBe(MAX_FONT_SIZE);
        expect(clampFontSize(18)).toBe(18);
    });

    it('falls back to the default for missing or invalid values', () => {
        expect(clampFontSize(undefined)).toBe(DEFAULT_FONT_SIZE);
        expect(clampFontSize(0)).toBe(DEFAULT_FONT_SIZE);
        expect(clampFontSize(Number.NaN)).toBe(DEFAULT_FONT_SIZE);
    });

    it('rounds fractional values', () => {
        expect(clampFontSize(15.6)).toBe(16);
    });
});
