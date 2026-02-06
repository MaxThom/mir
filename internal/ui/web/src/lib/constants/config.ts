/**
 * Application configuration constants
 */

export const APP_CONFIG = {
	NAME: 'Mir Cockpit',
	VERSION: '1.0.0',
	DESCRIPTION: 'Mir IoT Hub Management Interface'
} as const;

export const API_CONFIG = {
	BASE_URL: '/api',
	TIMEOUT: 30000, // 30 seconds
	RETRY_ATTEMPTS: 3
} as const;

export const PAGINATION_CONFIG = {
	DEFAULT_PAGE_SIZE: 20,
	PAGE_SIZE_OPTIONS: [10, 20, 50, 100]
} as const;
