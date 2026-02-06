import type { ApiResponse, ApiError } from '$lib/types';
import { API_CONFIG } from '$lib/constants';

/**
 * Base API client for making HTTP requests
 * Provides consistent error handling and response formatting
 */
class ApiClient {
	private baseUrl: string;
	private timeout: number;

	constructor(baseUrl: string = API_CONFIG.BASE_URL, timeout: number = API_CONFIG.TIMEOUT) {
		this.baseUrl = baseUrl;
		this.timeout = timeout;
	}

	private async request<T>(
		endpoint: string,
		options: RequestInit = {}
	): Promise<ApiResponse<T>> {
		const url = `${this.baseUrl}${endpoint}`;

		const controller = new AbortController();
		const timeoutId = setTimeout(() => controller.abort(), this.timeout);

		try {
			const response = await fetch(url, {
				...options,
				signal: controller.signal,
				headers: {
					'Content-Type': 'application/json',
					...options.headers
				}
			});

			clearTimeout(timeoutId);

			const data = await response.json();

			if (!response.ok) {
				const error: ApiError = {
					message: data.message || 'An error occurred',
					code: data.code || 'UNKNOWN_ERROR',
					status: response.status,
					details: data.details
				};
				throw error;
			}

			return {
				data,
				status: response.status,
				message: data.message
			};
		} catch (error) {
			clearTimeout(timeoutId);

			if (error instanceof Error && error.name === 'AbortError') {
				throw {
					message: 'Request timeout',
					code: 'TIMEOUT',
					status: 408
				} as ApiError;
			}

			throw error;
		}
	}

	async get<T>(endpoint: string, options?: RequestInit): Promise<ApiResponse<T>> {
		return this.request<T>(endpoint, {
			...options,
			method: 'GET'
		});
	}

	async post<T>(endpoint: string, data?: unknown, options?: RequestInit): Promise<ApiResponse<T>> {
		return this.request<T>(endpoint, {
			...options,
			method: 'POST',
			body: data ? JSON.stringify(data) : undefined
		});
	}

	async put<T>(endpoint: string, data?: unknown, options?: RequestInit): Promise<ApiResponse<T>> {
		return this.request<T>(endpoint, {
			...options,
			method: 'PUT',
			body: data ? JSON.stringify(data) : undefined
		});
	}

	async patch<T>(endpoint: string, data?: unknown, options?: RequestInit): Promise<ApiResponse<T>> {
		return this.request<T>(endpoint, {
			...options,
			method: 'PATCH',
			body: data ? JSON.stringify(data) : undefined
		});
	}

	async delete<T>(endpoint: string, options?: RequestInit): Promise<ApiResponse<T>> {
		return this.request<T>(endpoint, {
			...options,
			method: 'DELETE'
		});
	}
}

export const api = new ApiClient();
