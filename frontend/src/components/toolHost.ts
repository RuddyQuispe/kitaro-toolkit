import type { ToolCleanup, ToolContext, ToolDef } from '../tools/registry';

/**
 * Owns the content area the shell mounts tools into. Showing a tool first
 * runs the previous tool's cleanup (destroying its EditorViews, listeners,
 * etc.) so switching tools all day does not accumulate detached editors.
 */
export function createToolHost(content: HTMLElement) {
    let cleanup: ToolCleanup | void;

    return {
        show(tool: ToolDef, ctx: ToolContext): void {
            cleanup?.();
            cleanup = undefined;
            content.innerHTML = '';
            cleanup = tool.mount(content, ctx);
        },
    };
}
