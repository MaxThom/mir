import { IsMobile } from '../hooks/is-mobile.svelte';

/**
 * Sidebar store for managing sidebar state
 * Uses Svelte 5 runes for reactive state management
 */

class SidebarStore {
	isOpen = $state(true);
	#isMobile: IsMobile;

	constructor() {
		this.#isMobile = new IsMobile();
	}

	get isMobile(): boolean {
		return this.#isMobile.current;
	}

	toggle() {
		this.isOpen = !this.isOpen;
		this.persistState();
	}

	open() {
		this.isOpen = true;
		this.persistState();
	}

	close() {
		this.isOpen = false;
		this.persistState();
	}

	private persistState() {
		if (typeof window === 'undefined') return;
		localStorage.setItem('sidebar-open', JSON.stringify(this.isOpen));
	}

	initialize() {
		if (typeof window === 'undefined') return;

		// Load saved state
		const savedState = localStorage.getItem('sidebar-open');
		if (savedState !== null) {
			this.isOpen = JSON.parse(savedState);
		}

		// Auto-close on mobile
		if (this.isMobile) {
			this.isOpen = false;
		}
	}
}

export const sidebarStore = new SidebarStore();
