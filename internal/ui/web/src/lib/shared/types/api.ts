export type ApiResponse<T> = {
	data: T;
	error?: string;
	status: number;
	message?: string;
};

export type PaginatedResponse<T> = {
	items: T[];
	total: number;
	page: number;
	perPage: number;
	totalPages: number;
};

export type ApiError = {
	message: string;
	code: string;
	status: number;
	details?: Record<string, unknown>;
};

export type ContextsResponse = {
	currentContext: string;
	contexts: import('$lib/domains/contexts/types/types').Context[];
};
