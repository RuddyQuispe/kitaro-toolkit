// Ctrl + mouse wheel font zoom for every editor area (inputs, outputs, diff
// viewer). All editors share one size, exposed as the --editor-font-size CSS
// variable on :root. Bounds mirror internal/config (DefaultFontSize,
// MinFontSize, MaxFontSize).

export const DEFAULT_FONT_SIZE = 14;
export const MIN_FONT_SIZE = 10;
export const MAX_FONT_SIZE = 32;

const EDITOR_SELECTOR = '.tool-input, .tool-output, .diff-viewer';
const PERSIST_DELAY_MS = 400;

/** Bounds size to [MIN, MAX], rounding it; missing/invalid values fall back to the default. */
export function clampFontSize(size: number | undefined): number {
    if (size === undefined || !Number.isFinite(size) || size <= 0) return DEFAULT_FONT_SIZE;
    return Math.min(MAX_FONT_SIZE, Math.max(MIN_FONT_SIZE, Math.round(size)));
}

/** Wheel up (negative deltaY) grows by 1px, wheel down shrinks by 1px. */
export function nextFontSize(current: number, deltaY: number): number {
    return clampFontSize(current - Math.sign(deltaY));
}

let currentSize = DEFAULT_FONT_SIZE;

/** Applies size (clamped) to every editor via the shared CSS variable. */
export function applyFontSize(size: number | undefined): number {
    currentSize = clampFontSize(size);
    document.documentElement.style.setProperty('--editor-font-size', `${currentSize}px`);
    return currentSize;
}

let initialized = false;

/**
 * Installs the document-level Ctrl+wheel and Ctrl+0 listeners once. They
 * live on document, so they keep working across tool remounts. `persist`
 * is called (debounced) with the new size after each change.
 */
export function initFontZoom(persist: (size: number) => unknown): void {
    if (initialized) return;
    initialized = true;

    let timer: ReturnType<typeof setTimeout> | undefined;
    const update = (size: number) => {
        if (size === currentSize) return;
        applyFontSize(size);
        clearTimeout(timer);
        timer = setTimeout(() => {
            void Promise.resolve(persist(currentSize)).catch(() => {});
        }, PERSIST_DELAY_MS);
    };

    // passive: false so preventDefault() can stop the WebView's own page
    // zoom — but only for Ctrl+wheel over an editor, so normal scrolling and
    // Ctrl+wheel elsewhere keep their default behavior.
    document.addEventListener(
        'wheel',
        (e) => {
            if (!e.ctrlKey) return;
            const target = e.target as Element | null;
            if (!target?.closest?.(EDITOR_SELECTOR)) return;
            e.preventDefault();
            update(nextFontSize(currentSize, e.deltaY));
        },
        { passive: false },
    );

    document.addEventListener('keydown', (e) => {
        if (e.ctrlKey && e.key === '0') {
            e.preventDefault();
            update(DEFAULT_FONT_SIZE);
        }
    });
}
