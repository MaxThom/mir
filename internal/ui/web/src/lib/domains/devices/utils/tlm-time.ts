export const CHART_COLORS = [
	'var(--chart-1)',
	'var(--chart-2)',
	'var(--chart-3)',
	'var(--chart-4)',
	'var(--chart-5)'
] as const;

const DEVICE_VARIANTS = [
	(base: string) => base,
	(base: string) => `color-mix(in oklch, ${base} 60%, white)`,
	(base: string) => `color-mix(in oklch, ${base} 60%, black)`,
	(base: string) => `color-mix(in oklch, ${base} 35%, white)`,
	(base: string) => `color-mix(in oklch, ${base} 35%, black)`
];

export function getDeviceFieldColor(fieldIdx: number, deviceIdx: number): string {
	const base = CHART_COLORS[fieldIdx % CHART_COLORS.length];
	return DEVICE_VARIANTS[deviceIdx % DEVICE_VARIANTS.length](base);
}

export const MAX_AUTO_FIELDS = 5;

export type TimeFilter =
	| { mode: 'relative'; minutes: number }
	| { mode: 'absolute'; start: Date; end: Date };

export function getTimeRange(timeFilter: TimeFilter): { start: Date; end: Date } {
	if (timeFilter.mode === 'absolute') {
		const start = timeFilter.start;
		const end =
			timeFilter.end.getTime() <= start.getTime()
				? new Date(start.getTime() + 1000)
				: timeFilter.end;
		return { start, end };
	}
	const end = new Date();
	const start = new Date(end.getTime() - timeFilter.minutes * 60 * 1000);
	return { start, end };
}

export function getAggregationWindow(start: Date, end: Date): string | undefined {
	const hours = (end.getTime() - start.getTime()) / (1000 * 60 * 60);
	if (hours < 1) return undefined;
	if (hours < 6) return '10s';
	if (hours < 24) return '1m';
	if (hours < 168) return '10m';
	if (hours < 720) return '1h';
	return '6h';
}
