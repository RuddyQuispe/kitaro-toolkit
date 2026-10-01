import { jsonFormatterTool } from './JsonFormatter';
import { xmlFormatterTool } from './XmlFormatter';
import { jsonDiffTool } from './JsonDiff';
import { xmlDiffTool } from './XmlDiff';
import { sqlFormatterTool } from './SqlFormatter';
import { sqlDiffTool } from './SqlDiff';
import { jsonQueryToSqlTool } from './JsonQueryToSql';
import { sqlToJsonQueryTool } from './SqlToJsonQuery';
import { jsonToCsvTool } from './JsonToCsv';
import type { history } from '../../wailsjs/go/models';

export interface ToolDef {
    id: string;
    label: string;
    category?: string;
    /**
     * Renders the tool into `container`. Returns a cleanup function when the
     * tool holds resources that outlive its DOM (EditorViews, listeners on
     * window/document, timers); the shell calls it before mounting the next
     * tool. Tools with nothing to release may return nothing.
     */
    mount: (container: HTMLElement, ctx: ToolContext) => ToolCleanup | void;
}

/**
 * What the shell hands every tool on mount. Lets a tool open another tool
 * (e.g. History opening an entry) without importing the shell.
 */
export interface ToolContext {
    /** Switches to `toolId`, loading `restore` into it when given. */
    navigate(toolId: string, restore?: history.Entry): void;
    /** Whether `toolId` is a registered tool that navigate() can open. */
    hasTool(toolId: string): boolean;
    /** A full history entry to load on mount (load only, never re-run). */
    restore?: history.Entry;
}

export type ToolCleanup = () => void;

// Each tool registers itself here (id, tab label, its sidebar category, and
// a mount function that renders it into the given container). The shell
// never hardcodes a tool; it only groups by category, in order of first
// appearance in this array.
export const toolRegistry: ToolDef[] = [
    { ...jsonFormatterTool, category: 'JSON' },
    { ...xmlFormatterTool, category: 'XML' },
    { ...jsonDiffTool, category: 'JSON' },
    { ...xmlDiffTool, category: 'XML' },
    { ...sqlFormatterTool, category: 'SQL' },
    { ...sqlDiffTool, category: 'SQL' },
    { ...jsonQueryToSqlTool, category: 'JSON' },
    { ...sqlToJsonQueryTool, category: 'SQL' },
    { ...jsonToCsvTool, category: 'JSON' },
];
