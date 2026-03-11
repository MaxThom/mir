import { setMode } from 'mode-watcher';
import { browser } from '$app/environment';

export type Theme = 'light' | 'dark' | 'rust' | 'midnight';
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
		// midnight piggybacks on .dark so dark: Tailwind variants work;
		// .midnight CSS block comes after .dark in the cascade and overrides all variables
		if (theme === 'dark' || theme === 'midnight') {
			setMode('dark');
		} else {
			setMode('light');
		}
		document.documentElement.classList.remove('rust', 'midnight');
		if (theme === 'rust') {
			document.documentElement.classList.add('rust');
		} else if (theme === 'midnight') {
			document.documentElement.classList.add('midnight');
		}
	}
}

export const themeStore = new ThemeStore();
