import './shell.css';
import { toolRegistry, type ToolContext, type ToolDef } from '../tools/registry';
import { initTheme, toggleTheme, type Theme } from '../theme/useTheme';
import { initFontZoom } from './fontZoom';
import { createToolHost } from './toolHost';
import { SetFontSize } from '../../wailsjs/go/main/App';
import type { history } from '../../wailsjs/go/models';

export async function renderShell(root: HTMLElement) {
    root.innerHTML = `
        <div class="shell">
            <header class="shell-header">
                <span class="shell-title">Kitaro Toolkit</span>
                <button id="theme-toggle" class="btn">🌓</button>
            </header>
            <div class="shell-body">
                <aside id="shell-sidebar" class="shell-sidebar"></aside>
                <main id="shell-content" class="shell-content"></main>
            </div>
        </div>
    `;

    let theme: Theme = await initTheme();
    initFontZoom(SetFontSize);
    document.getElementById('theme-toggle')!.addEventListener('click', async () => {
        theme = await toggleTheme(theme);
    });

    const sidebar = document.getElementById('shell-sidebar')!;
    const content = document.getElementById('shell-content')!;

    if (toolRegistry.length === 0) {
        content.innerHTML = '<p class="empty-state">No tools registered yet.</p>';
        return;
    }

    // Preserves the order categories first appear in toolRegistry.
    const groups = new Map<string, ToolDef[]>();
    for (const tool of toolRegistry) {
        const category = tool.category ?? 'Other';
        const list = groups.get(category) ?? [];
        list.push(tool);
        groups.set(category, list);
    }

    const itemButtons: HTMLButtonElement[] = [];
    const buttonByToolId = new Map<string, HTMLButtonElement>();
    const host = createToolHost(content);

    function open(tool: ToolDef, restore?: history.Entry) {
        itemButtons.forEach((b) => b.classList.remove('active'));
        buttonByToolId.get(tool.id)?.classList.add('active');
        const ctx: ToolContext = {
            navigate(toolId, entry) {
                const target = toolRegistry.find((t) => t.id === toolId);
                if (target) open(target, entry);
            },
            hasTool: (toolId) => toolRegistry.some((t) => t.id === toolId),
            restore,
        };
        host.show(tool, ctx);
    }

    groups.forEach((tools, category) => {
        const groupEl = document.createElement('div');
        groupEl.className = 'sidebar-group';

        const header = document.createElement('button');
        header.className = 'sidebar-group-header';
        header.textContent = category;
        header.addEventListener('click', () => {
            groupEl.classList.toggle('collapsed');
        });
        groupEl.appendChild(header);

        const itemsEl = document.createElement('div');
        itemsEl.className = 'sidebar-group-items';

        tools.forEach((tool) => {
            const btn = document.createElement('button');
            btn.className = 'sidebar-item';
            btn.textContent = tool.label;
            btn.addEventListener('click', () => open(tool));
            itemsEl.appendChild(btn);
            itemButtons.push(btn);
            buttonByToolId.set(tool.id, btn);
        });

        groupEl.appendChild(itemsEl);
        sidebar.appendChild(groupEl);
    });

    itemButtons[0]?.click();
}
