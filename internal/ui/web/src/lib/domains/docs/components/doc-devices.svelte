<script lang="ts">
	import * as Sheet from '$lib/shared/components/shadcn/sheet/index.js';
	import * as Tabs from '$lib/shared/components/shadcn/tabs/index.js';
	import { Separator } from '$lib/shared/components/shadcn/separator/index.js';
	import CodeBlock from '$lib/shared/components/ui/code-block/code-block.svelte';
	import { docsStore } from '../stores/docs.svelte';
	import type { DocTab } from '../types/docs';
</script>

<Sheet.Header class="px-4 pt-4 pb-0">
	<Sheet.Title>List devices</Sheet.Title>
</Sheet.Header>
<Separator class="mt-3" />
<div class="bg-muted/50 px-4 py-3 text-sm leading-relaxed text-muted-foreground">
	The Devices page lists all registered IoT devices in your Mir instance. You can search, filter,
	and navigate to individual device details. Devices are the core entity in Mir — each represents a
	physical or virtual IoT endpoint.
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
				Use the search bar to filter devices by name or namespace. Click any row to open the device
				detail view. The status indicator shows real-time online/offline state. Use the "Create"
				button in the top-right to register a new device.
			</p>
		</Tabs.Content>
		<Tabs.Content value="cli">
			<div class="space-y-3">
				<div class="space-y-1">
					<CodeBlock title="List all devices" code="mir device list" lang="bash" />
				</div>
				<div class="space-y-1">
					<CodeBlock
						title="Create a device"
						code="mir device create --name my-device --namespace default"
						lang="bash"
					/>
				</div>
				<div class="space-y-1">
					<CodeBlock
						title="Delete a device"
						code="mir device delete --name my-device --namespace default"
						lang="bash"
					/>
				</div>
			</div>
		</Tabs.Content>
		<Tabs.Content value="gosdk">
			<CodeBlock
				title="List and create devices"
				code={`// List all devices
devices, err := mir.Client().ListDevices(ctx, &ListDevicesRequest{
    Targets: &Targets{},
})

// Create a device
device, err := mir.Client().CreateDevice(ctx, &CreateDeviceRequest{
    Device: &Device{
        Meta: &Meta{
            Name:      "my-device",
            Namespace: "default",
        },
    },
})`}
				lang="go"
			/>
		</Tabs.Content>
		<Tabs.Content value="tssdk">
			<CodeBlock
				title="List and create devices"
				code={`// List all devices
const devices = await m.client().listDevices({ targets: {} });

// Create a device
const device = await m.client().createDevice({
  device: {
    meta: { name: "my-device", namespace: "default" },
  },
});

// Delete a device
await m.client().deleteDevice({
  target: { name: "my-device", namespace: "default" },
});`}
				lang="typescript"
			/>
		</Tabs.Content>
	</div>
</Tabs.Root>
