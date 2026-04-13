import { dashboardApi, type Dashboard } from '$lib/domains/dashboards/api/dashboard-api';

class WelcomeStore {
	dashboard: Dashboard | null = $state(null);
	isLoading: boolean = $state(false);
	error: string | null = $state(null);

	async load(): Promise<void> {
		this.isLoading = true;
		this.error = null;
		try {
			this.dashboard = await dashboardApi.get('system', 'welcome');
		} catch (e) {
			this.error = e instanceof Error ? e.message : 'Failed to load welcome dashboard';
		} finally {
			this.isLoading = false;
		}
	}

	async refresh(): Promise<void> {
		return this.load();
	}
}

export const welcomeStore = new WelcomeStore();
