import { GetConfig, SetTheme } from '../../wailsjs/go/main/App';

export type Theme = 'dark' | 'light';

export async function initTheme(): Promise<Theme> {
    const cfg = await GetConfig();
    const theme = (cfg.theme as Theme) ?? 'dark';
    document.documentElement.setAttribute('data-theme', theme);
    return theme;
}

export async function toggleTheme(current: Theme): Promise<Theme> {
    const next: Theme = current === 'dark' ? 'light' : 'dark';
    document.documentElement.setAttribute('data-theme', next);
    await SetTheme(next);
    return next;
}
