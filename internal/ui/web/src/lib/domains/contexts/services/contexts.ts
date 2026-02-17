import { api } from '../../../shared/services/api';
import type { ApiResponse, ContextsResponse } from '../../../shared/types/api';

/**
 * Context service for fetching Mir contexts from the API
 * Provides methods to retrieve available contexts and current context
 */
export const contextService = {
	/**
	 * Get all contexts from the API
	 */
	async getAll(): Promise<ApiResponse<ContextsResponse>> {
		return api.get<ContextsResponse>('/v1/contexts');
	}
};
