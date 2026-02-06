/**
 * Application route constants
 * Centralized route definitions prevent magic strings and make refactoring easier
 */

export const ROUTES = {
	HOME: '/',
	DASHBOARD: '/dashboard',
	DEVICES: {
		LIST: '/devices',
		DETAIL: (id: string) => `/devices/${id}`,
		CREATE: '/devices/create'
	},
	SCHEMAS: {
		LIST: '/schemas',
		EXPLORER: '/schemas/explorer'
	},
	EVENTS: {
		LIST: '/events'
	},
	DOCS: {
		HOME: '/docs',
		INTRO: '/docs/introduction',
		GET_STARTED: '/docs/get-started',
		TUTORIALS: '/docs/tutorials',
		CHANGELOG: '/docs/changelog'
	},
	AUTH: {
		LOGIN: '/login',
		REGISTER: '/register',
		LOGOUT: '/logout'
	}
} as const;

export type RouteKey = keyof typeof ROUTES;
