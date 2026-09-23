import { jsonFormatterTool } from './JsonFormatter';
import { xmlFormatterTool } from './XmlFormatter';
import { jsonDiffTool } from './JsonDiff';
import { xmlDiffTool } from './XmlDiff';
import { sqlFormatterTool } from './SqlFormatter';
import { jsonToCsvTool } from './JsonToCsv';

export interface ToolDef {
    id: string;
    label: string;
    mount: (container: HTMLElement) => void;
}

// Each tool registers itself here (id, tab label, and a mount function that
// renders it into the given container). The shell never hardcodes a tool.
export const toolRegistry: ToolDef[] = [
    jsonFormatterTool,
    xmlFormatterTool,
    jsonDiffTool,
    xmlDiffTool,
    sqlFormatterTool,
    jsonToCsvTool,
];
