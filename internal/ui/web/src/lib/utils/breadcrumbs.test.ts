import { describe, it, expect } from 'vitest';
import { generateBreadcrumbs } from './breadcrumbs';

describe('generateBreadcrumbs', () => {
	it('should return home breadcrumb for root path', () => {
		const result = generateBreadcrumbs('/');
		expect(result).toEqual([{ label: 'Home', href: '/', isCurrentPage: true }]);
	});

	it('should generate breadcrumbs for single level path', () => {
		const result = generateBreadcrumbs('/devices');
		expect(result).toEqual([
			{ label: 'Home', href: '/', isCurrentPage: false },
			{ label: 'Devices', href: '/devices', isCurrentPage: true }
		]);
	});

	it('should generate breadcrumbs for multi-level path', () => {
		const result = generateBreadcrumbs('/devices/create');
		expect(result).toEqual([
			{ label: 'Home', href: '/', isCurrentPage: false },
			{ label: 'Devices', href: '/devices', isCurrentPage: false },
			{ label: 'Create', href: '/devices/create', isCurrentPage: true }
		]);
	});

	it('should handle kebab-case segments', () => {
		const result = generateBreadcrumbs('/schemas/schema-explorer');
		expect(result).toEqual([
			{ label: 'Home', href: '/', isCurrentPage: false },
			{ label: 'Schemas', href: '/schemas', isCurrentPage: false },
			{ label: 'Schema Explorer', href: '/schemas/schema-explorer', isCurrentPage: true }
		]);
	});

	it('should handle special cases (docs, api)', () => {
		const result = generateBreadcrumbs('/docs/intro');
		expect(result).toEqual([
			{ label: 'Home', href: '/', isCurrentPage: false },
			{ label: 'Documentation', href: '/docs', isCurrentPage: false },
			{ label: 'Intro', href: '/docs/intro', isCurrentPage: true }
		]);
	});

	it('should detect UUID as ID and label as Detail', () => {
		const result = generateBreadcrumbs('/devices/550e8400-e29b-41d4-a716-446655440000');
		expect(result).toEqual([
			{ label: 'Home', href: '/', isCurrentPage: false },
			{ label: 'Devices', href: '/devices', isCurrentPage: false },
			{
				label: 'Detail',
				href: '/devices/550e8400-e29b-41d4-a716-446655440000',
				isCurrentPage: true
			}
		]);
	});

	it('should detect numeric ID as Detail', () => {
		const result = generateBreadcrumbs('/devices/12345');
		expect(result).toEqual([
			{ label: 'Home', href: '/', isCurrentPage: false },
			{ label: 'Devices', href: '/devices', isCurrentPage: false },
			{ label: 'Detail', href: '/devices/12345', isCurrentPage: true }
		]);
	});

	it('should detect long alphanumeric ID as Detail', () => {
		const result = generateBreadcrumbs('/devices/abc123def456ghi');
		expect(result).toEqual([
			{ label: 'Home', href: '/', isCurrentPage: false },
			{ label: 'Devices', href: '/devices', isCurrentPage: false },
			{ label: 'Detail', href: '/devices/abc123def456ghi', isCurrentPage: true }
		]);
	});

	it('should handle trailing slash', () => {
		const result = generateBreadcrumbs('/devices/');
		expect(result).toEqual([
			{ label: 'Home', href: '/', isCurrentPage: false },
			{ label: 'Devices', href: '/devices', isCurrentPage: true }
		]);
	});

	it('should handle deep nesting with kebab-case', () => {
		const result = generateBreadcrumbs('/docs/tutorials/getting-started');
		expect(result).toEqual([
			{ label: 'Home', href: '/', isCurrentPage: false },
			{ label: 'Documentation', href: '/docs', isCurrentPage: false },
			{ label: 'Tutorials', href: '/docs/tutorials', isCurrentPage: false },
			{ label: 'Getting Started', href: '/docs/tutorials/getting-started', isCurrentPage: true }
		]);
	});
});
