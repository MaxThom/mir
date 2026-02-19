import { createColumnHelper, type FilterFn, type Row } from '@tanstack/table-core';
import type { Device } from '@mir/sdk';

export const deviceGlobalFilterFn: FilterFn<Device> = (
	row: Row<Device>,
	_columnId: string,
	filterValue: string
): boolean => {
	const search = filterValue.toLowerCase().trim();
	if (!search) return true;

	const name = (row.original.meta?.name ?? '').toLowerCase();
	const namespace = (row.original.meta?.namespace ?? '').toLowerCase();
	const deviceId = (row.original.spec?.deviceId ?? '').toLowerCase();
	const status = row.original.status?.online ? 'online' : 'offline';
	const disabled = row.original.spec?.disabled ? 'disabled' : 'enabled';
	const labels = Object.entries(row.original.meta?.labels ?? {})
		.map(([k, v]) => `${k}=${v}`)
		.join(' ')
		.toLowerCase();
	const hbSeconds = row.original.status?.lastHearthbeat?.seconds;
	const heartbeat = hbSeconds
		? new Date(Number(hbSeconds) * 1000).toLocaleString().toLowerCase()
		: '';

	return (
		name.includes(search) ||
		namespace.includes(search) ||
		deviceId.includes(search) ||
		status.includes(search) ||
		disabled.includes(search) ||
		labels.includes(search) ||
		heartbeat.includes(search)
	);
};
deviceGlobalFilterFn.autoRemove = (val: string) => !val || val.trim() === '';

const col = createColumnHelper<Device>();

export const deviceColumns = [
	col.accessor((d) => d.meta?.name ?? '—', {
		id: 'name',
		header: 'Name',
	}),
	col.accessor((d) => d.meta?.namespace ?? '—', {
		id: 'namespace',
		header: 'Namespace',
	}),
	col.accessor((d) => d.spec?.deviceId ?? '—', {
		id: 'deviceId',
		header: 'Device ID',
	}),
	col.accessor((d) => d.status?.online ?? false, {
		id: 'status',
		header: 'Status',
	}),
	col.accessor((d) => d.status?.lastHearthbeat, {
		id: 'lastHeartbeat',
		header: 'Last Heartbeat',
	}),
	col.accessor((d) => d.meta?.labels ?? {}, {
		id: 'labels',
		header: 'Labels',
		sortingFn: (rowA, rowB) => {
			const a = Object.entries(rowA.original.meta?.labels ?? {})
				.map(([k, v]) => `${k}=${v}`)
				.join(',');
			const b = Object.entries(rowB.original.meta?.labels ?? {})
				.map(([k, v]) => `${k}=${v}`)
				.join(',');
			return a.localeCompare(b);
		},
	}),
	col.display({
		id: 'actions',
		header: '',
		enableSorting: false,
		enableGlobalFilter: false,
	}),
];
