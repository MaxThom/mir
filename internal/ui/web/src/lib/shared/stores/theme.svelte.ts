import { setMode } from 'mode-watcher';
import { browser } from '$app/environment';

export type Theme = 'light' | 'dark' | 'rust' | 'midnight' | 'hacker' | 'mocha';
const STORAGE_KEY = 'mir-theme';

class ThemeStore {
	current = $state<Theme>('light');
	private committed: Theme = 'light';

	init() {
		if (!browser) return;
		const saved = localStorage.getItem(STORAGE_KEY) as Theme | null;
		if (saved) {
			this.committed = saved;
			this.apply(saved);
		}
	}

	set(theme: Theme) {
		this.committed = theme;
		this.apply(theme);
		if (browser) localStorage.setItem(STORAGE_KEY, theme);
	}

	preview(theme: Theme) {
		this.apply(theme);
	}

	revert() {
		this.apply(this.committed);
	}

	private apply(theme: Theme) {
		this.current = theme;
		if (!browser) return;
		if (theme === 'dark' || theme === 'midnight' || theme === 'hacker' || theme === 'mocha') {
			setMode('dark');
		} else {
			setMode('light');
		}
		document.documentElement.classList.remove('rust', 'midnight', 'hacker', 'mocha');
		if (theme === 'rust') {
			document.documentElement.classList.add('rust');
		} else if (theme === 'midnight') {
			document.documentElement.classList.add('midnight');
		} else if (theme === 'hacker') {
			document.documentElement.classList.add('hacker');
		} else if (theme === 'mocha') {
			document.documentElement.classList.add('mocha');
		}
	}
}

export const themeStore = new ThemeStore();
