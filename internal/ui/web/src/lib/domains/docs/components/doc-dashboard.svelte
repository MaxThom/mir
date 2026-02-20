<script lang="ts">
	import * as Sheet from '$lib/components/ui/sheet/index.js';
	import * as Tabs from '$lib/components/ui/tabs/index.js';
	import { Separator } from '$lib/components/ui/separator/index.js';
	import CodeBlock from '$lib/components/ui/code-block/code-block.svelte';
	import { docsStore } from '../stores/docs.svelte';
	import type { DocTab } from '../types/docs';
</script>

<Sheet.Header class="px-4 pt-4 pb-0">
	<Sheet.Title>Dashboard</Sheet.Title>
</Sheet.Header>
<Separator class="mt-3" />
<div class="bg-muted/50 px-4 py-3 text-sm leading-relaxed text-muted-foreground">
	The Dashboard gives you a high-level overview of your Mir IoT Hub instance. View connected device
	counts, recent events, and system health at a glance.
</div>
<Separator />
<Tabs.Root
	value={docsStore.activeTab}
	onValueChange={(v) => docsStore.setTab(v as DocTab)}
	class="flex flex-1 flex-col overflow-hidden"
>
	<Tabs.List class="h-auto w-full gap-0 border-b border-border bg-transparent p-0">
		<Tabs.Trigger value="web" class="h-auto flex-1 border-b-2 border-transparent px-3 py-2.5"
			>Web</Tabs.Trigger
		>
		<Tabs.Trigger value="cli" class="h-auto flex-1 border-b-2 border-transparent px-3 py-2.5"
			>CLI</Tabs.Trigger
		>
		<Tabs.Trigger value="gosdk" class="h-auto flex-1 border-b-2 border-transparent px-3 py-2.5"
			>Go</Tabs.Trigger
		>
		<Tabs.Trigger value="tssdk" class="h-auto flex-1 border-b-2 border-transparent px-3 py-2.5"
			>TypeScript</Tabs.Trigger
		>
	</Tabs.List>
	<div class="flex-1 overflow-y-auto px-4 py-4">
		<Tabs.Content value="web">
			<p class="text-sm leading-relaxed text-muted-foreground">
				The Dashboard displays live device counts, connection status, and a feed of recent system
				events. Cards update automatically as devices connect or disconnect. Click any device count
				card to jump directly to the filtered device list.
			</p>
		</Tabs.Content>
		<Tabs.Content value="cli">
			<div class="space-y-3">
				<div class="space-y-1">
					<CodeBlock title="Show Mir server status" code="mir status" lang="bash" />
				</div>
				<div class="space-y-1">
					<CodeBlock title="List all context" code="mir context list" lang="bash" />
				</div>
			</div>
		</Tabs.Content>
		<Tabs.Content value="gosdk">
			<CodeBlock
				title="Connect to Mir"
				code={`// Connect to Mir
mir, err := mir.Connect("my-module", natsConn)
if err != nil {
    log.Fatal(err)
}
defer mir.Disconnect()

// List devices
devices, err := mir.Client().ListDevices(ctx, &ListDevicesRequest{})
if err != nil {
    log.Fatal(err)
}`}
				lang="go"
			/>
		</Tabs.Content>
		<Tabs.Content value="tssdk">
			<CodeBlock
				title="Connect to Mir"
				code={`// Connect to Mir
const m = await Mir.connect("my-module", ["ws://localhost:9222"], jwt, nkey);

// List devices
const devices = await m.client().listDevices({});

await m.disconnect();`}
				lang="typescript"
			/>
		</Tabs.Content>
	</div>
</Tabs.Root>
