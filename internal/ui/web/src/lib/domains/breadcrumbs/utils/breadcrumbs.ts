/**
 * Breadcrumb utilities for generating navigation breadcrumbs from routes
 */

export type Breadcrumb = {
	label: string;
	href: string;
	isCurrentPage?: boolean;
};

/**
 * Generate breadcrumbs from a pathname
 * @param pathname - The current page pathname (e.g., "/devices/create")
 * @returns Array of breadcrumb objects
 */
export function generateBreadcrumbs(pathname: string): Breadcrumb[] {
	// Handle root path
	if (pathname === '/') {
		return [{ label: 'Dashboard', href: '/', isCurrentPage: true }];
	}

	// Split path and filter empty segments
	const segments = pathname.split('/').filter(Boolean);

	// Always start with home (not current page since we have more segments)
	const breadcrumbs: Breadcrumb[] = [];

	// Build breadcrumbs for each segment
	let currentPath = '';
	segments.forEach((segment, index) => {
		currentPath += `/${segment}`;
		const isLast = index === segments.length - 1;

		breadcrumbs.push({
			label: formatSegment(segment),
			href: currentPath,
			isCurrentPage: isLast
		});
	});

	return breadcrumbs;
}

/**
 * Format a path segment into a readable label
 * @param segment - Path segment (e.g., "devices", "create", "abc-123")
 * @returns Formatted label
 */
function formatSegment(segment: string): string {
	// Handle special cases
	const specialCases: Record<string, string> = {
		docs: 'Documentation',
		api: 'API'
	};

	if (specialCases[segment]) {
		return specialCases[segment];
	}

	// Check if it looks like an ID (UUID, nanoid, etc.)
	if (isLikelyId(segment)) {
		return 'Detail';
	}

	// Convert kebab-case or snake_case to Title Case
	return segment
		.split(/[-_]/)
		.map((word) => word.charAt(0).toUpperCase() + word.slice(1))
		.join(' ');
}

/**
 * Check if a segment looks like an ID
 * @param segment - Path segment to check
 * @returns true if segment appears to be an ID
 */
function isLikelyId(segment: string): boolean {
	// UUID pattern (e.g., 550e8400-e29b-41d4-a716-446655440000)
	if (/^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$/i.test(segment)) {
		return true;
	}

	// Long alphanumeric without hyphens (likely nanoid or similar)
	// Must be all lowercase or all uppercase, 12+ chars, no hyphens
	if (/^[a-z0-9]{12,}$/i.test(segment) && !segment.includes('-') && !segment.includes('_')) {
		return true;
	}

	// Numeric ID
	if (/^\d+$/.test(segment)) {
		return true;
	}

	return false;
}
