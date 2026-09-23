import { jsonFormatterTool } from './JsonFormatter';

export interface ToolDef {
    id: string;
    label: string;
    mount: (container: HTMLElement) => void;
}

// Each tool registers itself here (id, tab label, and a mount function that
// renders it into the given container). The shell never hardcodes a tool.
export const toolRegistry: ToolDef[] = [jsonFormatterTool];
