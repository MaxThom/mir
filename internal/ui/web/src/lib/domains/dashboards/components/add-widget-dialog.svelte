<script lang="ts">
	import { dashboardStore } from '../stores/dashboard.svelte';
	import { deviceStore } from '$lib/domains/devices/stores/device.svelte';
	import { mirStore } from '$lib/domains/mir/stores/mir.svelte';
	import DeviceTargetBuilder from './device-target-builder.svelte';
	import TlmTimeRangePicker from '$lib/domains/devices/components/telemetry/tlm-time-range-picker.svelte';
	import * as Sheet from '$lib/shared/components/shadcn/sheet';
	import { Button } from '$lib/shared/components/shadcn/button';
	import { Input } from '$lib/shared/components/shadcn/input';
	import { Spinner } from '$lib/shared/components/shadcn/spinner';
	import type {
		WidgetType,
		DeviceTargetConfig,
		TelemetryWidgetConfig,
		EventsWidgetConfig,
		CommandWidgetConfig,
		ConfigWidgetConfig
	} from '../api/dashboard-api';
	import type { TimeFilter } from '$lib/domains/devices/utils/tlm-time';
	import type { TelemetryGroup } from '@mir/sdk';
	import { DeviceTarget } from '@mir/sdk';
	import ActivityIcon from '@lucide/svelte/icons/activity';
	import TerminalIcon from '@lucide/svelte/icons/terminal';
	import SlidersHorizontalIcon from '@lucide/svelte/icons/sliders-horizontal';
	import CalendarClockIcon from '@lucide/svelte/icons/calendar-clock';

	let { open = $bindable(false) }: { open?: boolean } = $props();

	$effect(() => {
		if (open && mirStore.mir && deviceStore.devices.length === 0 && !deviceStore.isLoading) {
			deviceStore.loadDevices(mirStore.mir);
		}
	});

	type Step = 'type' | 'target' | 'config';

	let step = $state<Step>('type');
	let selectedType = $state<WidgetType | null>(null);
	let title = $state('');
	let target = $state<DeviceTargetConfig>({});

	// Telemetry config
	let measurements = $state<{ name: string; fields: string[] }[]>([]);
	let measurementsLoading = $state(false);
	let selectedMeasurement = $state('');
	let timeFilter = $state<TimeFilter>({ mode: 'relative', minutes: 60 });

	// Events config
	let eventLimit = $state(50);

	$effect(() => {
		if (step === 'config' && selectedType === 'telemetry' && mirStore.mir) {
			loadMeasurements();
		}
	});

	async function loadMeasurements() {
		measurementsLoading = true;
		measurements = [];
		try {
			// Find a representative device: prefer explicit IDs, fall back to first dynamic match
			let firstId: string | undefined;
			if (target.ids?.length) {
				firstId = target.ids[0];
			} else {
				const match = deviceStore.devices.find((d) => {
					const nsMatch =
						!target.namespaces?.length ||
						target.namespaces.includes(d.meta?.namespace ?? 'default');
					const labelMatch =
						!target.labels ||
						Object.entries(target.labels).every(([k, v]) => d.meta?.labels?.[k] === v);
					return nsMatch && labelMatch;
				});
				firstId = match?.spec?.deviceId;
			}
			if (!firstId) return;

			const deviceTarget = new DeviceTarget({ ids: [firstId] });
			const groups = (await mirStore.mir!.client().listTelemetry().request(deviceTarget)) as TelemetryGroup[];
			measurements = groups
				.flatMap((g) => g.descriptors ?? [])
				.map((d) => ({ name: d.name ?? '', fields: d.fields ?? [] }))
				.filter((d) => d.name);
		} catch {
			measurements = [];
		} finally {
			measurementsLoading = false;
		}
	}

	function reset() {
		step = 'type';
		selectedType = null;
		title = '';
		target = {};
		measurements = [];
		measurementsLoading = false;
		selectedMeasurement = '';
		timeFilter = { mode: 'relative', minutes: 60 };
		eventLimit = 50;
	}

	function selectType(t: WidgetType) {
		selectedType = t;
		title = typeLabel(t);
		step = 'target';
	}

	function typeLabel(t: WidgetType): string {
		switch (t) {
			case 'telemetry': return 'Telemetry';
			case 'command': return 'Command';
			case 'config': return 'Configuration';
			case 'events': return 'Events';
		}
	}

	function goToConfig() {
		step = 'config';
	}

	async function addWidget() {
		if (!dashboardStore.activeDashboard || !selectedType) return;

		let config: TelemetryWidgetConfig | CommandWidgetConfig | ConfigWidgetConfig | EventsWidgetConfig;
		switch (selectedType) {
			case 'telemetry': {
				const descriptor = measurements.find((m) => m.name === selectedMeasurement);
				const minutes =
					timeFilter.mode === 'relative'
						? timeFilter.minutes
						: Math.round((timeFilter.end.getTime() - timeFilter.start.getTime()) / 60000);
				config = {
					target,
					measurement: selectedMeasurement,
					fields: descriptor?.fields ?? [],
					timeMinutes: minutes
				} satisfies TelemetryWidgetConfig;
				break;
			}
			case 'command':
				config = { target } satisfies CommandWidgetConfig;
				break;
			case 'config':
				config = { target } satisfies ConfigWidgetConfig;
				break;
			case 'events':
				config = { target, limit: eventLimit } satisfies EventsWidgetConfig;
				break;
		}

		await dashboardStore.addWidget(dashboardStore.activeDashboard, selectedType, title, config);
		open = false;
		reset();
	}
</script>

<Sheet.Root bind:open onOpenChange={(o) => { if (!o) reset(); }}>
	<Sheet.Content side="right" class="w-[48rem]">
		<Sheet.Header>
			<Sheet.Title>Add Widget</Sheet.Title>
			<Sheet.Description>
				{#if step === 'type'}Step 1 of 3 — Choose widget type{/if}
				{#if step === 'target'}Step 2 of 3 — Select devices{/if}
				{#if step === 'config'}Step 3 of 3 — Configure widget{/if}
			</Sheet.Description>
		</Sheet.Header>

		<div class="mt-4 space-y-4 px-1">
			<!-- Step 1: Type -->
			{#if step === 'type'}
				<div class="grid grid-cols-2 gap-2">
					{#each [
						{ type: 'telemetry' as WidgetType, icon: ActivityIcon, label: 'Telemetry' },
						{ type: 'command' as WidgetType, icon: TerminalIcon, label: 'Command' },
						{ type: 'config' as WidgetType, icon: SlidersHorizontalIcon, label: 'Configuration' },
						{ type: 'events' as WidgetType, icon: CalendarClockIcon, label: 'Events' }
					] as item (item.type)}
						<button
							class="border-border hover:border-primary flex flex-col items-center gap-2 rounded-lg border p-4 transition-colors"
							onclick={() => selectType(item.type)}
						>
							<item.icon class="h-6 w-6" />
							<span class="text-sm">{item.label}</span>
						</button>
					{/each}
				</div>
			{/if}

			<!-- Step 2: Target devices -->
			{#if step === 'target'}
				<div class="space-y-2">
					<label for="widget-title" class="text-sm font-medium">Title</label>
					<Input id="widget-title" bind:value={title} placeholder="Widget title" />
				</div>

				<DeviceTargetBuilder
					devices={deviceStore.devices}
					isLoading={deviceStore.isLoading}
					bind:target
				/>

				<div class="flex gap-2">
					<Button variant="outline" onclick={() => (step = 'type')}>Back</Button>
					<Button
						onclick={goToConfig}
						disabled={selectedType !== 'events' && target.ids !== undefined && target.ids.length === 0}
					>
						Next
					</Button>
				</div>
			{/if}

			<!-- Step 3: Type-specific config -->
			{#if step === 'config'}
				{#if selectedType === 'telemetry'}
					<div class="space-y-3">
						<div class="space-y-1">
							<p class="text-sm font-medium">Measurement</p>
							{#if measurementsLoading}
								<div class="flex items-center gap-2 py-3 text-sm text-muted-foreground">
									<Spinner class="h-4 w-4" />
									<span>Loading measurements…</span>
								</div>
							{:else if measurements.length === 0}
								<p class="py-2 text-sm text-muted-foreground">No measurements found for selected device.</p>
							{:else}
								<div class="border-border max-h-52 divide-y overflow-y-auto rounded-md border">
									{#each measurements as m (m.name)}
										<button
											onclick={() => (selectedMeasurement = m.name)}
											class="flex w-full items-center gap-3 px-3 py-2 text-left text-sm transition-colors hover:bg-accent
												{selectedMeasurement === m.name ? 'bg-muted' : ''}"
										>
											<span
												class="flex h-4 w-4 shrink-0 items-center justify-center rounded-full border
													{selectedMeasurement === m.name ? 'border-primary bg-primary' : 'border-muted-foreground'}"
											>
												{#if selectedMeasurement === m.name}
													<span class="h-1.5 w-1.5 rounded-full bg-primary-foreground"></span>
												{/if}
											</span>
											<span class="flex-1 font-mono">{m.name}</span>
											<span class="text-xs text-muted-foreground"
												>{m.fields.length} field{m.fields.length !== 1 ? 's' : ''}</span
											>
										</button>
									{/each}
								</div>
							{/if}
						</div>
						<div class="space-y-1">
							<p class="text-sm font-medium">Time range</p>
							<TlmTimeRangePicker bind:timeFilter />
						</div>
					</div>
				{:else if selectedType === 'events'}
					<div class="space-y-1">
						<label for="widget-maxevents" class="text-sm font-medium">Max events</label>
						<Input id="widget-maxevents" type="number" bind:value={eventLimit} min={1} max={500} />
					</div>
				{:else}
					<p class="text-muted-foreground text-sm">
						{typeLabel(selectedType!)} widget is ready to add. Select a command or config after adding.
					</p>
				{/if}

				<div class="flex gap-2">
					<Button variant="outline" onclick={() => (step = 'target')}>Back</Button>
					<Button
						onclick={addWidget}
						disabled={dashboardStore.isSaving || (selectedType === 'telemetry' && !selectedMeasurement)}
					>
						{dashboardStore.isSaving ? 'Adding…' : 'Add Widget'}
					</Button>
				</div>
			{/if}
		</div>
	</Sheet.Content>
</Sheet.Root>
