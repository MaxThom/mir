<script lang="ts">
	import type { Cell, Row } from '@tanstack/table-core';
	import ActivityIcon from '@lucide/svelte/icons/activity';
	import TerminalIcon from '@lucide/svelte/icons/terminal';
	import SettingsIcon from '@lucide/svelte/icons/settings';
	import ListIcon from '@lucide/svelte/icons/list';
	import BracesIcon from '@lucide/svelte/icons/braces';
	import * as Table from '$lib/components/ui/table';
	import * as Tooltip from '$lib/components/ui/tooltip';
	import * as ButtonGroup from '$lib/components/ui/button-group';
	import { Badge } from '$lib/components/ui/badge';
	import { Button } from '$lib/components/ui/button';
	import { cn } from '$lib/utils';
	import type { Device } from '@mir/sdk';
	import { relativeTime, formatFullDate } from '$lib/shared/utils/time';

	const DEVICE_ACTIONS = [
		{ icon: ActivityIcon, label: 'Telemetry', path: 'telemetry' },
		{ icon: TerminalIcon, label: 'Commands', path: 'commands' },
		{ icon: SettingsIcon, label: 'Configuration', path: 'configuration' },
		{ icon: ListIcon, label: 'Events', path: 'events' },
		{ icon: BracesIcon, label: 'Schema', path: 'schema' }
	] as const;

	let { cell, row }: { cell: Cell<Device, unknown>; row: Row<Device> } = $props();
</script>

<Table.Cell class={cell.column.id === 'actions' ? 'w-px pr-2 whitespace-nowrap' : ''}>
	{#if cell.column.id === 'name'}
		<a href="/devices/{row.original.spec?.deviceId}" class="font-medium hover:underline">
			{cell.getValue() ?? '—'}
		</a>
	{:else if cell.column.id === 'namespace'}
		<Badge variant="outline" class="font-mono text-xs font-normal">
			{cell.getValue() ?? '—'}
		</Badge>
	{:else if cell.column.id === 'deviceId'}
		<span class="font-mono text-xs text-muted-foreground">
			{cell.getValue() ?? '—'}
		</span>
	{:else if cell.column.id === 'labels'}
		<div class="flex flex-col gap-1">
			{#each Object.entries(cell.getValue() as Record<string, string>) as [k, v] (k)}
				<Badge variant="secondary" class="font-mono text-xs font-normal">{k}={v}</Badge>
			{/each}
		</div>
	{:else if cell.column.id === 'status'}
		<div class="flex items-center gap-2">
			<span
				class={cn(
					'h-2 w-2 shrink-0 rounded-full',
					cell.getValue()
						? 'bg-emerald-500 shadow-[0_0_0_3px_--theme(--color-emerald-500/0.2)]'
						: 'bg-muted-foreground/30'
				)}
			></span>
			<span
				class={cn(
					'text-sm',
					cell.getValue()
						? 'font-medium text-emerald-600 dark:text-emerald-400'
						: 'text-muted-foreground'
				)}
			>
				{cell.getValue() ? 'Online' : 'Offline'}
			</span>
			{#if row.original.spec?.disabled}
				<Badge variant="destructive" class="text-xs">Disabled</Badge>
			{/if}
		</div>
	{:else if cell.column.id === 'lastHeartbeat'}
		{#if cell.getValue()}
			<Tooltip.Root>
				<Tooltip.Trigger
					class="cursor-default text-sm text-muted-foreground underline decoration-dotted underline-offset-2 hover:text-foreground"
				>
					{relativeTime((cell.getValue() as { seconds: bigint | number }).seconds)}
				</Tooltip.Trigger>
				<Tooltip.Content side="left">
					{formatFullDate((cell.getValue() as { seconds: bigint | number }).seconds)}
				</Tooltip.Content>
			</Tooltip.Root>
		{:else}
			<span class="text-muted-foreground">—</span>
		{/if}
	{:else if cell.column.id === 'actions'}
		{@const deviceId = row.original.spec?.deviceId ?? ''}
		<ButtonGroup.Root>
			{#each DEVICE_ACTIONS as action (action.path)}
				<Tooltip.Root>
					<Tooltip.Trigger>
						{#snippet child({ props })}
							<Button
								{...props}
								variant="ghost"
								size="icon-sm"
								disabled={!deviceId}
								href="/devices/{deviceId}/{action.path}"
							>
								<action.icon class="h-3.5 w-3.5" />
							</Button>
						{/snippet}
					</Tooltip.Trigger>
					<Tooltip.Content>{action.label}</Tooltip.Content>
				</Tooltip.Root>
			{/each}
		</ButtonGroup.Root>
	{:else}
		{cell.getValue() ?? '—'}
	{/if}
</Table.Cell>
