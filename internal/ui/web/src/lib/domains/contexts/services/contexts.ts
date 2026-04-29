import { api } from '../../../shared/services/api';
import type { ApiResponse, ContextsResponse, CredentialsResponse } from '../../../shared/types/api';

export const contextService = {
	async getAll(): Promise<ApiResponse<ContextsResponse>> {
		return api.get<ContextsResponse>('/v1/contexts');
	},

	async getCredentials(contextName: string, password?: string | null): Promise<string | null> {
		const url = `/api/v1/credentials?context=${encodeURIComponent(contextName)}`;
		const body = JSON.stringify({ password: password ?? '' });
		const response = await fetch(url, {
			method: 'POST',
			headers: { 'Content-Type': 'application/json' },
			body
		});

		if (response.status === 204) return null;

		if (!response.ok) {
			const text = await response.text();
			const err = new Error(text.trim() || response.statusText);
			(err as Error & { status: number }).status = response.status;
			throw err;
		}

		const data: CredentialsResponse = await response.json();
		return data.creds;
	}
};
