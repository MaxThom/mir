import type { User } from '../types/user';

/**
 * User store for managing authentication state
 * Uses Svelte 5 runes for reactive state management
 */
class UserStore {
	user = $state<User | null>(null);
	isLoading = $state(false);
	error = $state<string | null>(null);

	get isAuthenticated(): boolean {
		return this.user !== null;
	}

	get isAdmin(): boolean {
		return this.user?.role === 'admin';
	}

	async login(email: string, password: string) {
		this.isLoading = true;
		this.error = null;

		try {
			// TODO: Replace with actual API call
			const response = await fetch('/api/auth/login', {
				method: 'POST',
				headers: { 'Content-Type': 'application/json' },
				body: JSON.stringify({ email, password })
			});

			if (!response.ok) {
				throw new Error('Login failed');
			}

			this.user = await response.json();
		} catch (err) {
			this.error = err instanceof Error ? err.message : 'Login failed';
			throw err;
		} finally {
			this.isLoading = false;
		}
	}

	logout() {
		this.user = null;
		this.error = null;
	}

	setUser(user: User) {
		this.user = user;
		this.error = null;
	}
}

export const userStore = new UserStore();
