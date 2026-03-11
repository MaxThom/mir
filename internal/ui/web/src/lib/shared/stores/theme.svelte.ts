import { setMode } from 'mode-watcher';
import { browser } from '$app/environment';

export type Theme = 'light' | 'dark' | 'rust';
const STORAGE_KEY = 'mir-theme';

class ThemeStore {
	current = $state<Theme>('light');

	init() {
		if (!browser) return;
		const saved = localStorage.getItem(STORAGE_KEY) as Theme | null;
		if (saved) this.apply(saved);
	}

	set(theme: Theme) {
		this.apply(theme);
		if (browser) localStorage.setItem(STORAGE_KEY, theme);
	}

	private apply(theme: Theme) {
		this.current = theme;
		if (!browser) return;
		if (theme === 'dark') {
			setMode('dark');
			document.documentElement.classList.remove('rust');
		} else {
			setMode('light');
			if (theme === 'rust') {
				document.documentElement.classList.add('rust');
			} else {
				document.documentElement.classList.remove('rust');
			}
		}
	}
}

export const themeStore = new ThemeStore();
