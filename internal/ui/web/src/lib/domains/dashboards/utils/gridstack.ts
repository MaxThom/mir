import type { GridStack, GridStackWidget } from 'gridstack';
import type { Widget } from '../api/dashboard-api';

export function widgetsToGridItems(widgets: Widget[]): GridStackWidget[] {
	return widgets.map((w) => ({
		id: w.id,
		x: w.x,
		y: w.y,
		w: w.w,
		h: w.h
	}));
}

export function serializeLayout(grid: GridStack): Pick<Widget, 'id' | 'x' | 'y' | 'w' | 'h'>[] {
	const items = grid.save(false) as GridStackWidget[];
	return items.map((item) => ({
		id: item.id as string,
		x: item.x ?? 0,
		y: item.y ?? 0,
		w: item.w ?? 4,
		h: item.h ?? 4
	}));
}
