<script lang="ts">
	import { onMount, onDestroy, tick } from 'svelte';
	import { SvelteMap } from 'svelte/reactivity';
	import type { GridStack } from 'gridstack';
	import { dashboardStore } from '../stores/dashboard.svelte';
	import type { Widget } from '../api/dashboard-api';
	import WidgetWrapper from './widget-wrapper.svelte';
	import WidgetTelemetry from './widget-telemetry.svelte';
	import WidgetCommand from './widget-command.svelte';
	import WidgetConfig from './widget-config.svelte';
	import WidgetEvents from './widget-events.svelte';
	import WidgetDevice from './widget-device.svelte';
	import WidgetDevicePills from './widget-device-pills.svelte';
	import type { TelemetryWidgetConfig, CommandWidgetConfig, ConfigWidgetConfig, EventsWidgetConfig, DeviceWidgetConfig } from '../api/dashboard-api';

	let { widgets, refreshTick = 0, onEditWidget }: { widgets: Widget[]; refreshTick?: number; onEditWidget?: (w: Widget) => void } = $props();

	const widgetDevices = new SvelteMap<string, { id: string; name: string; color: string }[]>();

	let gridEl: HTMLDivElement | undefined;
	let grid: GridStack | undefined;

	onMount(async () => {
		const { GridStack } = await import('gridstack');
		await import('gridstack/dist/gridstack.min.css');

		grid = GridStack.init(
			{
				cellHeight: 80,
				margin: 4,
				handle: '.grid-stack-item-content-drag-handle',
				float: false
			},
			gridEl!
		);

		grid.on('change', () => {
			if (!dashboardStore.activeDashboard || !grid) return;
			if (!dashboardStore.editMode && !dashboardStore.isCreatingNew) return;
			const items = grid.save(false) as import('gridstack').GridStackWidget[];
			const layout = items.map((item) => ({
				id: item.id as string,
				x: item.x ?? 0,
				y: item.y ?? 0,
				w: item.w ?? 4,
				h: item.h ?? 4
			}));
			dashboardStore.saveLayout(dashboardStore.activeDashboard, layout);
		});
	});

	onDestroy(() => {
		grid?.destroy(false);
	});

	$effect(() => {
		const editMode = dashboardStore.editMode || dashboardStore.isCreatingNew;
		if (!grid) return;
		tick().then(() => {
			editMode ? grid!.enable() : grid!.disable();
		});
	});

	// Register any widgets added dynamically (GridStack only sees elements present at init time)
	$effect(() => {
		void widgets;
		if (!grid || !gridEl) return;
		tick().then(() => {
			gridEl!
				.querySelectorAll<HTMLElement>('.grid-stack-item')
				.forEach((el) => {
					if (!(el as HTMLElement & { gridstackNode?: unknown }).gridstackNode) {
						grid!.makeWidget(el);
					}
				});
		});
	});
</script>

<div class="grid-stack" class:gs-edit-mode={dashboardStore.editMode || dashboardStore.isCreatingNew} bind:this={gridEl}>
	{#each widgets as widget (widget.id)}
		<div
			class="grid-stack-item"
			{...{ 'gs-id': widget.id, 'gs-x': widget.x, 'gs-y': widget.y, 'gs-w': widget.w, 'gs-h': widget.h }}
		>
			<div class="grid-stack-item-content">
				{#snippet headerPills()}
					<WidgetDevicePills
						devices={widgetDevices.get(widget.id) ?? []}
						target={widget.config.target}
					/>
				{/snippet}
				<WidgetWrapper
					title={widget.title}
					editMode={dashboardStore.editMode || dashboardStore.isCreatingNew}
					onEdit={() => onEditWidget?.(widget)}
					onRemove={() =>
						dashboardStore.activeDashboard &&
						dashboardStore.removeWidget(dashboardStore.activeDashboard, widget.id)}
					headerExtra={headerPills}
				>
					{#if widget.type === 'telemetry'}
						<WidgetTelemetry
							widgetId={widget.id}
							config={widget.config as TelemetryWidgetConfig}
							{refreshTick}
							onDevicesReady={(infos) => widgetDevices.set(widget.id, infos)}
						/>
					{:else if widget.type === 'command'}
						<WidgetCommand
							widgetId={widget.id}
							config={widget.config as CommandWidgetConfig}
							onDevicesReady={(infos) => widgetDevices.set(widget.id, infos)}
						/>
					{:else if widget.type === 'config'}
						<WidgetConfig
							widgetId={widget.id}
							config={widget.config as ConfigWidgetConfig}
							{refreshTick}
							onDevicesReady={(infos) => widgetDevices.set(widget.id, infos)}
						/>
					{:else if widget.type === 'events'}
						<WidgetEvents config={widget.config as EventsWidgetConfig} {refreshTick} />
					{:else if widget.type === 'device'}
						<WidgetDevice
							widgetId={widget.id}
							config={widget.config as DeviceWidgetConfig}
							{refreshTick}
							onDevicesReady={(infos) => widgetDevices.set(widget.id, infos)}
						/>
					{/if}
				</WidgetWrapper>
			</div>
		</div>
	{/each}
</div>

<style>
	/* Hide GridStack resize handles when not in edit mode */
	:global(.grid-stack:not(.gs-edit-mode) .ui-resizable-handle) {
		display: none !important;
	}
	/* Ensure resize handles appear above widget card content */
	:global(.grid-stack.gs-edit-mode .ui-resizable-handle) {
		z-index: 10;
	}
</style>
