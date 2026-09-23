import { toolRegistry } from '../tools/registry';
import { initTheme, toggleTheme, type Theme } from '../theme/useTheme';

export async function renderShell(root: HTMLElement) {
    root.innerHTML = `
        <div class="shell">
            <header class="shell-header">
                <span class="shell-title">kitaro-rq</span>
                <button id="theme-toggle" class="btn">🌓</button>
            </header>
            <nav id="shell-tabs" class="shell-tabs"></nav>
            <main id="shell-content" class="shell-content"></main>
        </div>
    `;

    let theme: Theme = await initTheme();
    document.getElementById('theme-toggle')!.addEventListener('click', async () => {
        theme = await toggleTheme(theme);
    });

    const tabs = document.getElementById('shell-tabs')!;
    const content = document.getElementById('shell-content')!;

    if (toolRegistry.length === 0) {
        content.innerHTML = '<p class="empty-state">No tools registered yet.</p>';
        return;
    }

    toolRegistry.forEach((tool, i) => {
        const btn = document.createElement('button');
        btn.className = 'tab-btn';
        btn.textContent = tool.label;
        btn.addEventListener('click', () => {
            content.innerHTML = '';
            tool.mount(content);
        });
        tabs.appendChild(btn);
        if (i === 0) btn.click();
    });
}
