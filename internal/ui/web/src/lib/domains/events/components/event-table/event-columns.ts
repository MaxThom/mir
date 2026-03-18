import { createColumnHelper, type FilterFn, type Row } from '@tanstack/table-core';
import type { MirEvent } from '@mir/sdk';

export const eventGlobalFilterFn: FilterFn<MirEvent> = (
	row: Row<MirEvent>,
	_columnId: string,
	filterValue: string
): boolean => {
	const search = filterValue.toLowerCase().trim();
	if (!search) return true;

	const type = (row.original.spec?.type ?? '').toLowerCase();
	const reason = (row.original.spec?.reason ?? '').toLowerCase();
	const message = (row.original.spec?.message ?? '').toLowerCase();
	const device = (row.original.spec?.relatedObject?.meta?.name ?? '').toLowerCase();

	return type.includes(search) || reason.includes(search) || message.includes(search) || device.includes(search);
};
eventGlobalFilterFn.autoRemove = (val: string) => !val || val.trim() === '';

const col = createColumnHelper<MirEvent>();

export const eventColumns = [
	col.display({
		id: 'expand',
		header: '',
		enableSorting: false,
		enableGlobalFilter: false,
	}),
	col.accessor((e) => e.spec?.relatedObject?.meta?.name ?? '', {
		id: 'deviceName',
		header: 'Device',
		enableSorting: true,
	}),
	col.accessor((e) => e.spec?.type ?? 'normal', {
		id: 'type',
		header: 'Type',
		enableSorting: true,
		enableGlobalFilter: false,
	}),
	col.accessor((e) => e.spec?.reason ?? '', {
		id: 'reason',
		header: 'Reason',
		enableSorting: true,
	}),
	col.accessor((e) => e.spec?.message ?? '', {
		id: 'message',
		header: 'Message',
		enableSorting: false,
	}),
	col.accessor((e) => e.status?.lastAt, {
		id: 'lastAt',
		header: 'Last seen',
		enableSorting: true,
		sortingFn: (rowA, rowB) => {
			const a = rowA.original.status?.lastAt?.getTime() ?? 0;
			const b = rowB.original.status?.lastAt?.getTime() ?? 0;
			return a - b;
		},
	}),
];
