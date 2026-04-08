<script lang="ts">
	import { dashboardStore } from '../stores/dashboard.svelte';
	import { deviceStore } from '$lib/domains/devices/stores/device.svelte';
	import { mirStore } from '$lib/domains/mir/stores/mir.svelte';
	import DeviceTargetBuilder from './device-target-builder.svelte';
	import * as Dialog from '$lib/shared/components/shadcn/dialog';
	import { Button } from '$lib/shared/components/shadcn/button';
	import { Input } from '$lib/shared/components/shadcn/input';
	import { Spinner } from '$lib/shared/components/shadcn/spinner';
	import { Badge } from '$lib/shared/components/shadcn/badge';
	import type {
		Widget,
		WidgetType,
		DeviceTargetConfig,
		TelemetryWidgetConfig,
		EventsWidgetConfig,
		CommandWidgetConfig,
		ConfigWidgetConfig,
		DeviceWidgetConfig,
		TextWidgetConfig
	} from '../api/dashboard-api';
	import type { TelemetryGroup, CommandGroup, ConfigGroup } from '@mir/sdk';
	import { DeviceTarget } from '@mir/sdk';
	import ActivityIcon from '@lucide/svelte/icons/activity';
	import TerminalIcon from '@lucide/svelte/icons/terminal';
	import SlidersHorizontalIcon from '@lucide/svelte/icons/sliders-horizontal';
	import CalendarClockIcon from '@lucide/svelte/icons/calendar-clock';
	import InfoIcon from '@lucide/svelte/icons/info';
	import CpuIcon from '@lucide/svelte/icons/cpu';
	import PieChartIcon from '@lucide/svelte/icons/pie-chart';
	import FileTextIcon from '@lucide/svelte/icons/file-text';

	let {
		open = $bindable(false),
		editWidget = $bindable<Widget | null>(null)
	}: { open?: boolean; editWidget?: Widget | null } = $props();

	$effect(() => {
		if (open && mirStore.mir && deviceStore.devices.length === 0 && !deviceStore.isLoading) {
			deviceStore.loadDevices(mirStore.mir);
		}
	});

	// Pre-populate state when opening in edit mode
	$effect(() => {
		if (open && editWidget) {
			selectedType = editWidget.type;
			title = editWidget.title;
			step = 'target';
			if (editWidget.type === 'telemetry') {
				const c = editWidget.config as TelemetryWidgetConfig;
				selectedMeasurement = c.measurement;
			} else if (editWidget.type === 'command') {
				const c = editWidget.config as CommandWidgetConfig;
				selectedCommandName = c.selectedCommand ?? '';
			} else if (editWidget.type === 'config') {
				const c = editWidget.config as ConfigWidgetConfig;
				selectedConfigName = c.selectedConfig ?? '';
			} else if (editWidget.type === 'events') {
				eventLimit = (editWidget.config as EventsWidgetConfig).limit;
			} else if (editWidget.type === 'device') {
				const c = editWidget.config as DeviceWidgetConfig;
				selectedDeviceView = c.view ?? 'info';
			} else if (editWidget.type === 'text') {
				textContent = (editWidget.config as TextWidgetConfig).content ?? '';
			}
		}
	});

	type Step = 'type' | 'target' | 'config';

	type MeasurementGroup = {
		devices: { id: string; name: string; namespace: string }[];
		measurements: { name: string; fields: string[] }[];
	};

	let step = $state<Step>('type');
	let selectedType = $state<WidgetType | null>(null);
	let title = $state('');
	let target = $state<DeviceTargetConfig>({});

	// Telemetry config
	let measurementGroups = $state<MeasurementGroup[]>([]);
	let measurementsLoading = $state(false);
	let selectedMeasurement = $state('');

	// Command config
	let commandGroups = $state<CommandGroup[]>([]);
	let commandsLoading = $state(false);
	let selectedCommandName = $state('');

	// Config config
	let configGroups = $state<ConfigGroup[]>([]);
	let configsLoading = $state(false);
	let selectedConfigName = $state('');

	// Events config
	let eventLimit = $state(50);

	// Device config
	let selectedDeviceView = $state<'info' | 'properties' | 'status'>('info');

	// Text config
	let textContent = $state('');

	$effect(() => {
		if (step === 'config' && selectedType === 'telemetry' && mirStore.mir) {
			loadMeasurements();
		}
	});

	$effect(() => {
		if (step === 'config' && selectedType === 'command' && mirStore.mir) {
			loadCommandsForWizard();
		}
	});

	$effect(() => {
		if (step === 'config' && selectedType === 'config' && mirStore.mir) {
			loadConfigsForWizard();
		}
	});

	async function loadMeasurements() {
		measurementsLoading = true;
		measurementGroups = [];
		try {
			let deviceIds: string[] = [];
			if (target.ids?.length) {
				deviceIds = target.ids;
			} else {
				deviceIds = deviceStore.devices
					.filter((d) => {
						const nsMatch =
							!target.namespaces?.length ||
							target.namespaces.includes(d.meta?.namespace ?? 'default');
						const labelMatch =
							!target.labels ||
							Object.entries(target.labels).every(([k, v]) => d.meta?.labels?.[k] === v);
						return nsMatch && labelMatch;
					})
					.map((d) => d.spec?.deviceId)
					.filter((id): id is string => Boolean(id));
			}
			if (!deviceIds.length) return;

			const deviceTarget = new DeviceTarget({ ids: deviceIds });
			const groups = (await mirStore
				.mir!.client()
				.listTelemetry()
				.request(deviceTarget)) as TelemetryGroup[];
			measurementGroups = groups.map((g) => ({
				devices: g.ids ?? [],
				measurements: (g.descriptors ?? [])
					.map((d) => ({ name: d.name ?? '', fields: d.fields ?? [] }))
					.filter((d) => d.name)
			}));
		} catch {
			measurementGroups = [];
		} finally {
			measurementsLoading = false;
		}
	}

	async function loadCommandsForWizard() {
		commandsLoading = true;
		commandGroups = [];
		try {
			let deviceIds: string[] = [];
			if (target.ids?.length) {
				deviceIds = target.ids;
			} else {
				deviceIds = deviceStore.devices
					.filter((d) => {
						const nsMatch =
							!target.namespaces?.length ||
							target.namespaces.includes(d.meta?.namespace ?? 'default');
						const labelMatch =
							!target.labels ||
							Object.entries(target.labels).every(([k, v]) => d.meta?.labels?.[k] === v);
						return nsMatch && labelMatch;
					})
					.map((d) => d.spec?.deviceId)
					.filter((id): id is string => Boolean(id));
			}
			if (!deviceIds.length) return;
			const deviceTarget = new DeviceTarget({ ids: deviceIds });
			commandGroups = await mirStore.mir!.client().listCommands().request(deviceTarget);
		} catch {
			commandGroups = [];
		} finally {
			commandsLoading = false;
		}
	}

	async function loadConfigsForWizard() {
		configsLoading = true;
		configGroups = [];
		try {
			let deviceIds: string[] = [];
			if (target.ids?.length) {
				deviceIds = target.ids;
			} else {
				deviceIds = deviceStore.devices
					.filter((d) => {
						const nsMatch =
							!target.namespaces?.length ||
							target.namespaces.includes(d.meta?.namespace ?? 'default');
						const labelMatch =
							!target.labels ||
							Object.entries(target.labels).every(([k, v]) => d.meta?.labels?.[k] === v);
						return nsMatch && labelMatch;
					})
					.map((d) => d.spec?.deviceId)
					.filter((id): id is string => Boolean(id));
			}
			if (!deviceIds.length) return;
			const deviceTarget = new DeviceTarget({ ids: deviceIds });
			configGroups = await mirStore.mir!.client().listConfigs().request(deviceTarget);
		} catch {
			configGroups = [];
		} finally {
			configsLoading = false;
		}
	}

	function reset() {
		step = 'type';
		selectedType = null;
		title = '';
		target = {};
		measurementGroups = [];
		measurementsLoading = false;
		selectedMeasurement = '';
		commandGroups = [];
		commandsLoading = false;
		selectedCommandName = '';
		configGroups = [];
		configsLoading = false;
		selectedConfigName = '';
		eventLimit = 50;
		selectedDeviceView = 'info';
		textContent = '';
		editWidget = null;
	}

	function selectType(t: WidgetType) {
		selectedType = t;
		title = typeLabel(t);
		step = 'target';
	}

	function typeLabel(t: WidgetType): string {
		switch (t) {
			case 'telemetry':
				return 'Telemetry';
			case 'command':
				return 'Command';
			case 'config':
				return 'Configuration';
			case 'events':
				return 'Events';
			case 'device':
				return 'Device';
			case 'text':
				return 'Text';
		}
	}

	function goToConfig() {
		step = 'config';
	}

	function buildConfig(): TelemetryWidgetConfig | CommandWidgetConfig | ConfigWidgetConfig | EventsWidgetConfig | DeviceWidgetConfig | TextWidgetConfig {
		switch (selectedType!) {
			case 'telemetry': {
				const descriptor = measurementGroups
					.flatMap((g) => g.measurements)
					.find((m) => m.name === selectedMeasurement);
				return {
					target,
					measurement: selectedMeasurement,
					fields: descriptor?.fields ?? (editWidget?.config as TelemetryWidgetConfig)?.fields ?? [],
					timeMinutes: 60
				} satisfies TelemetryWidgetConfig;
			}
			case 'command':
				return { target, selectedCommand: selectedCommandName } satisfies CommandWidgetConfig;
			case 'config':
				return { target, selectedConfig: selectedConfigName } satisfies ConfigWidgetConfig;
			case 'events':
				return { target, limit: eventLimit } satisfies EventsWidgetConfig;
			case 'device':
				return { target, view: selectedDeviceView } satisfies DeviceWidgetConfig;
			case 'text':
				return { content: textContent } satisfies TextWidgetConfig;
		}
	}

	async function saveWidget() {
		if (!dashboardStore.activeDashboard || !selectedType) return;
		const config = buildConfig();
		if (editWidget) {
			const w = dashboardStore.activeDashboard.spec.widgets.find((w) => w.id === editWidget!.id);
			if (w) w.title = title;
			dashboardStore.saveWidgetConfig(dashboardStore.activeDashboard, editWidget.id, config);
		} else {
			await dashboardStore.addWidget(dashboardStore.activeDashboard, selectedType, title, config);
		}
		open = false;
		reset();
	}
</script>

<Dialog.Root
	bind:open
	onOpenChange={(o) => {
		if (!o) reset();
	}}
>
	<Dialog.Content class="h-[90vh] max-h-[90vh] w-[90vw] max-w-[90vw] overflow-y-auto content-start">
		<Dialog.Header>
			<Dialog.Title>{editWidget ? 'Edit Widget' : 'Add Widget'}</Dialog.Title>
			<Dialog.Description>
				{#if step === 'type'}Step 1 of {selectedType === 'text' ? '2' : '3'} — Choose widget type{/if}
				{#if step === 'target'}{editWidget ? 'Step 1 of 2' : selectedType === 'text' ? 'Step 2 of 2' : 'Step 2 of 3'} — {selectedType === 'text' ? 'Name your widget' : 'Select devices'}{/if}
				{#if step === 'config'}{editWidget ? 'Step 2 of 2' : 'Step 3 of 3'} — Configure widget{/if}
			</Dialog.Description>
		</Dialog.Header>

		<div class="flex h-full flex-col space-y-4 px-2">
			<!-- Step 1: Type -->
			{#if step === 'type'}
				<div class="flex flex-1 items-start justify-center">
					<div class="grid w-full max-w-2xl grid-cols-3 gap-4">
						{#each [{ type: 'telemetry' as WidgetType, icon: ActivityIcon, label: 'Telemetry', desc: 'Visualize time-series data from device sensors' }, { type: 'command' as WidgetType, icon: TerminalIcon, label: 'Command', desc: 'Send commands and view responses from devices' }, { type: 'config' as WidgetType, icon: SlidersHorizontalIcon, label: 'Configuration', desc: 'Manage and push configuration to devices' }, { type: 'events' as WidgetType, icon: CalendarClockIcon, label: 'Events', desc: 'Monitor events and audit logs from the fleet' }, { type: 'device' as WidgetType, icon: CpuIcon, label: 'Device', desc: 'View device meta, status and properties' }, { type: 'text' as WidgetType, icon: FileTextIcon, label: 'Text', desc: 'Markdown content for notes and documentation' }] as item (item.type)}
							<button
								class="flex flex-col items-center gap-4 rounded-xl border border-border p-10 text-center transition-colors hover:border-primary hover:bg-accent"
								onclick={() => selectType(item.type)}
							>
								<item.icon class="h-12 w-12 text-muted-foreground" />
								<div>
									<p class="font-semibold">{item.label}</p>
									<p class="mt-1 text-xs text-muted-foreground">{item.desc}</p>
								</div>
							</button>
						{/each}
					</div>
				</div>
			{/if}

			<!-- Step 2: Target devices -->
			{#if step === 'target'}
				<div class="space-y-2">
					<label for="widget-title" class="text-sm font-medium">Title</label>
					<Input id="widget-title" bind:value={title} placeholder="Widget title" />
				</div>

				{#if selectedType !== 'text'}
					<DeviceTargetBuilder
						devices={deviceStore.devices}
						isLoading={deviceStore.isLoading}
						bind:target
						initialTarget={(editWidget?.config as any)?.target}
					/>
				{/if}

				<div class="flex gap-2">
					<Button variant="outline" onclick={() => editWidget ? (open = false, reset()) : (step = 'type')}>{editWidget ? 'Cancel' : 'Back'}</Button>
					{#if selectedType === 'text'}
						<Button onclick={saveWidget} disabled={dashboardStore.isSaving}>
							{dashboardStore.isSaving ? (editWidget ? 'Saving…' : 'Adding…') : (editWidget ? 'Save' : 'Add Widget')}
						</Button>
					{:else}
						<Button
							onclick={goToConfig}
							disabled={selectedType !== 'events' &&
								!target.ids?.length &&
								!target.namespaces?.length &&
								!Object.keys(target.labels ?? {}).length}
						>
							Next
						</Button>
					{/if}
				</div>
			{/if}

			<!-- Step 3: Type-specific config -->
			{#if step === 'config'}
				<div class="space-y-3">
					{#if selectedType === 'telemetry'}
						<div class="space-y-1">
							<p class="text-sm font-medium">Measurement</p>
							{#if measurementsLoading}
								<div class="flex items-center gap-2 py-3 text-sm text-muted-foreground">
									<Spinner class="h-4 w-4" />
									<span>Loading measurements…</span>
								</div>
							{:else if measurementGroups.length === 0}
								<p class="py-2 text-sm text-muted-foreground">
									No measurements found for selected devices.
								</p>
							{:else}
								<div class="max-h-64 space-y-2 overflow-y-auto">
									{#each measurementGroups as group, gi (gi)}
										<div class="rounded-md border border-border">
											<div class="flex flex-wrap items-center gap-1 border-b px-3 py-2">
												{#each group.devices as dev (dev.id)}
													<Badge variant="secondary" class="font-mono text-xs font-normal"
														>{dev.name}</Badge
													>
												{/each}
											</div>
											<div class="divide-y">
												{#each group.measurements as m (m.name)}
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
										</div>
									{/each}
								</div>
							{/if}
						</div>
						{#if measurementGroups.length > 1}
							<div
								class="flex items-start gap-2 rounded-md border border-border bg-muted/40 px-3 py-2 text-xs text-muted-foreground"
							>
								<InfoIcon class="mt-0.5 h-3.5 w-3.5 shrink-0" />
								<span
									>Selected devices have different schemas. Only devices that support the chosen
									measurement will contribute data to this widget.</span
								>
							</div>
						{/if}
					{:else if selectedType === 'command'}
						<div class="space-y-1">
							<p class="text-sm font-medium">Command</p>
							{#if commandsLoading}
								<div class="flex items-center gap-2 py-3 text-sm text-muted-foreground">
									<Spinner class="h-4 w-4" />
									<span>Loading commands…</span>
								</div>
							{:else if commandGroups.length === 0}
								<p class="py-2 text-sm text-muted-foreground">
									No commands found for selected devices.
								</p>
							{:else}
								<div class="max-h-64 space-y-2 overflow-y-auto">
									{#each commandGroups as group, gi (gi)}
										<div class="rounded-md border border-border">
											<div class="flex flex-wrap items-center gap-1 border-b px-3 py-2">
												{#each group.ids as dev (dev.id)}
													<Badge variant="secondary" class="font-mono text-xs font-normal"
														>{dev.name}</Badge
													>
												{/each}
											</div>
											<div class="divide-y">
												{#each group.descriptors as cmd (cmd.name)}
													<button
														onclick={() => (selectedCommandName = cmd.name)}
														class="flex w-full items-center gap-3 px-3 py-2 text-left text-sm transition-colors hover:bg-accent
															{selectedCommandName === cmd.name ? 'bg-muted' : ''}"
													>
														<span
															class="flex h-4 w-4 shrink-0 items-center justify-center rounded-full border
																{selectedCommandName === cmd.name ? 'border-primary bg-primary' : 'border-muted-foreground'}"
														>
															{#if selectedCommandName === cmd.name}
																<span class="h-1.5 w-1.5 rounded-full bg-primary-foreground"></span>
															{/if}
														</span>
														<span class="flex-1 font-mono">{cmd.name}</span>
													</button>
												{/each}
											</div>
										</div>
									{/each}
								</div>
							{/if}
						</div>
					{:else if selectedType === 'config'}
						<div class="space-y-1">
							<p class="text-sm font-medium">Configuration</p>
							{#if configsLoading}
								<div class="flex items-center gap-2 py-3 text-sm text-muted-foreground">
									<Spinner class="h-4 w-4" />
									<span>Loading configurations…</span>
								</div>
							{:else if configGroups.length === 0}
								<p class="py-2 text-sm text-muted-foreground">
									No configurations found for selected devices.
								</p>
							{:else}
								<div class="max-h-64 space-y-2 overflow-y-auto">
									{#each configGroups as group, gi (gi)}
										<div class="rounded-md border border-border">
											<div class="flex flex-wrap items-center gap-1 border-b px-3 py-2">
												{#each group.ids as dev (dev.id)}
													<Badge variant="secondary" class="font-mono text-xs font-normal"
														>{dev.name}</Badge
													>
												{/each}
											</div>
											<div class="divide-y">
												{#each group.descriptors as cfg (cfg.name)}
													<button
														onclick={() => (selectedConfigName = cfg.name)}
														class="flex w-full items-center gap-3 px-3 py-2 text-left text-sm transition-colors hover:bg-accent
															{selectedConfigName === cfg.name ? 'bg-muted' : ''}"
													>
														<span
															class="flex h-4 w-4 shrink-0 items-center justify-center rounded-full border
																{selectedConfigName === cfg.name ? 'border-primary bg-primary' : 'border-muted-foreground'}"
														>
															{#if selectedConfigName === cfg.name}
																<span class="h-1.5 w-1.5 rounded-full bg-primary-foreground"></span>
															{/if}
														</span>
														<span class="flex-1 font-mono">{cfg.name}</span>
													</button>
												{/each}
											</div>
										</div>
									{/each}
								</div>
							{/if}
						</div>
					{:else if selectedType === 'events'}
						<div class="space-y-1">
							<label for="widget-maxevents" class="text-sm font-medium">Max events</label>
							<Input id="widget-maxevents" type="number" bind:value={eventLimit} min={1} max={500} />
						</div>
					{:else if selectedType === 'device'}
						<div class="space-y-1">
							<p class="text-sm font-medium">View</p>
							<div class="grid grid-cols-3 gap-3">
								<button
									onclick={() => (selectedDeviceView = 'info')}
									class="flex flex-col items-center gap-2 rounded-xl border p-6 text-center transition-colors hover:border-primary hover:bg-accent {selectedDeviceView === 'info' ? 'border-primary bg-accent' : 'border-border'}"
								>
									<InfoIcon class="h-8 w-8 text-muted-foreground" />
									<div>
										<p class="font-semibold">Info</p>
										<p class="mt-0.5 text-xs text-muted-foreground">Meta, Spec &amp; Status</p>
									</div>
								</button>
								<button
									onclick={() => (selectedDeviceView = 'properties')}
									class="flex flex-col items-center gap-2 rounded-xl border p-6 text-center transition-colors hover:border-primary hover:bg-accent {selectedDeviceView === 'properties' ? 'border-primary bg-accent' : 'border-border'}"
								>
									<SlidersHorizontalIcon class="h-8 w-8 text-muted-foreground" />
									<div>
										<p class="font-semibold">Properties</p>
										<p class="mt-0.5 text-xs text-muted-foreground">Desired &amp; Reported</p>
									</div>
								</button>
								<button
									onclick={() => (selectedDeviceView = 'status')}
									class="flex flex-col items-center gap-2 rounded-xl border p-6 text-center transition-colors hover:border-primary hover:bg-accent {selectedDeviceView === 'status' ? 'border-primary bg-accent' : 'border-border'}"
								>
									<PieChartIcon class="h-8 w-8 text-muted-foreground" />
									<div>
										<p class="font-semibold">Status</p>
										<p class="mt-0.5 text-xs text-muted-foreground">Online &amp; Offline</p>
									</div>
								</button>
							</div>
						</div>
					{:else}
						<p class="text-sm text-muted-foreground">
							{editWidget ? typeLabel(selectedType!) + ' widget is ready. Click Save to update.' : typeLabel(selectedType!) + ' widget is ready to add.'}
						</p>
					{/if}
				</div>

				<div class="flex gap-2">
					<Button variant="outline" onclick={() => (step = 'target')}>Back</Button>
					<Button
						onclick={saveWidget}
						disabled={dashboardStore.isSaving ||
							(selectedType === 'telemetry' && !selectedMeasurement) ||
							(selectedType === 'command' && !selectedCommandName) ||
							(selectedType === 'config' && !selectedConfigName)}
					>
						{dashboardStore.isSaving ? (editWidget ? 'Saving…' : 'Adding…') : (editWidget ? 'Save' : 'Add Widget')}
					</Button>
				</div>
			{/if}
		</div>
	</Dialog.Content>
</Dialog.Root>
