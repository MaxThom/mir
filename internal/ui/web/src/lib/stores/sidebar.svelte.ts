/**
 * Sidebar store for managing sidebar state
 * Uses Svelte 5 runes for reactive state management
 */

class SidebarStore {
	isOpen = $state(true);
	isMobile = $state(false);

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

	setMobile(mobile: boolean) {
		this.isMobile = mobile;
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

		// Detect mobile
		const checkMobile = () => {
			this.isMobile = window.innerWidth < 768;
			if (this.isMobile) {
				this.isOpen = false;
			}
		};

		checkMobile();
		window.addEventListener('resize', checkMobile);
	}
}

export const sidebarStore = new SidebarStore();
