import type { Context } from '../types/types';
import { contextService } from '../services/contexts';

/**
 * Context store for managing Mir server contexts
 * Uses Svelte 5 runes for reactive state management
 * Handles loading contexts from API and session-based active context selection
 */
class ContextStore {
	contexts = $state<Context[]>([]);
	activeContext = $state<Context | null>(null);
	isLoading = $state(true);
	error = $state<string | null>(null);

	get hasContexts(): boolean {
		return this.contexts.length > 0;
	}

	get isReady(): boolean {
		return !this.isLoading && this.activeContext !== null;
	}

	/**
	 * Initialize the store by fetching contexts from the API
	 * Determines active context from sessionStorage or API's currentContext
	 */
	async initialize() {
		this.isLoading = true;
		this.error = null;

		try {
			const response = await contextService.getAll();
			this.contexts = response.data.contexts;

			// Determine active context:
			// 1. Check sessionStorage for previously selected context
			const savedName = sessionStorage.getItem('mir-active-context');
			const savedContext = this.contexts.find((c) => c.name === savedName);

			if (savedContext) {
				this.activeContext = savedContext;
			} else {
				// 2. Fallback to API's currentContext
				const apiCurrent = this.contexts.find(
					(c) => c.name === response.data.currentContext
				);
				this.activeContext = apiCurrent || this.contexts[0] || null;
				this.persistActiveContext();
			}
		} catch (err) {
			this.error = err instanceof Error ? err.message : 'Failed to load contexts';
			this.contexts = [];
			this.activeContext = null;
		} finally {
			this.isLoading = false;
		}
	}

	/**
	 * Set the active context by name
	 * Updates session storage for persistence across page refreshes
	 */
	setActiveContext(contextName: string) {
		const context = this.contexts.find((c) => c.name === contextName);
		if (context) {
			this.activeContext = context;
			this.persistActiveContext();
		}
	}

	/**
	 * Persist the active context to session storage
	 */
	private persistActiveContext() {
		if (this.activeContext) {
			sessionStorage.setItem('mir-active-context', this.activeContext.name);
		}
	}
}

export const contextStore = new ContextStore();
