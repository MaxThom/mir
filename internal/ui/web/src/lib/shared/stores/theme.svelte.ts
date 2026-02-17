/**
 * Theme store for managing light/dark/system theme preference
 * Uses Svelte 5 runes for reactive state management
 */

export type Theme = 'light' | 'dark' | 'system';

class ThemeStore {
	current = $state<Theme>('system');

	get isDark(): boolean {
		if (this.current === 'system') {
			if (typeof window !== 'undefined') {
				return window.matchMedia('(prefers-color-scheme: dark)').matches;
			}
			return false;
		}
		return this.current === 'dark';
	}

	get resolvedTheme(): 'light' | 'dark' {
		return this.isDark ? 'dark' : 'light';
	}

	toggle() {
		this.current = this.current === 'light' ? 'dark' : 'light';
		this.applyTheme();
	}

	setTheme(theme: Theme) {
		this.current = theme;
		this.applyTheme();
	}

	private applyTheme() {
		if (typeof window === 'undefined') return;

		const root = document.documentElement;
		const theme = this.resolvedTheme;

		root.classList.remove('light', 'dark');
		root.classList.add(theme);
		localStorage.setItem('theme', this.current);
	}

	initialize() {
		if (typeof window === 'undefined') return;

		// Load saved preference
		const savedTheme = localStorage.getItem('theme') as Theme | null;
		if (savedTheme) {
			this.current = savedTheme;
		}

		// Apply initial theme
		this.applyTheme();

		// Listen for system theme changes
		window.matchMedia('(prefers-color-scheme: dark)').addEventListener('change', () => {
			if (this.current === 'system') {
				this.applyTheme();
			}
		});
	}
}

export const themeStore = new ThemeStore();
