<script lang="ts">
	import * as Sheet from '$lib/shared/components/shadcn/sheet/index.js';
	import * as Tabs from '$lib/shared/components/shadcn/tabs/index.js';
	import { Separator } from '$lib/shared/components/shadcn/separator/index.js';
	import CodeBlock from '$lib/shared/components/ui/code-block/code-block.svelte';
	import { docsStore } from '../stores/docs.svelte';
	import type { DocTab } from '../types/docs';
</script>

<Sheet.Header class="px-4 pt-4 pb-0">
	<Sheet.Title>Create a device</Sheet.Title>
</Sheet.Header>
<Separator class="mt-3" />
<div class="bg-muted/50 px-4 py-3 text-sm leading-relaxed text-muted-foreground">
	Register a new device. The device ID must match what is configured on the physical device.
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
			<div class="space-y-4">
				<div>
					<p class="mb-2 text-xs font-semibold tracking-wide text-muted-foreground uppercase">
						Create Many
					</p>
					<p class="text-sm text-muted-foreground">
						<span class="font-medium text-foreground">Create</span> navigates to the new device after
						creation. <span class="font-medium text-foreground">Create Many</span> stays on the page so
						you can register multiple devices in a row.
					</p>
				</div>
				<div>
					<p class="mb-2 text-xs font-semibold tracking-wide text-muted-foreground uppercase">
						YAML / JSON editor
					</p>
					<p class="text-sm text-muted-foreground">
						The file icon switches to a code editor pre-filled from the current form values. Edit as
						YAML or JSON and hit <span class="font-medium text-foreground">Create</span> — the form
						fields are updated from the parsed content before submitting.
					</p>
				</div>
			</div>
		</Tabs.Content>
		<Tabs.Content value="cli">
			<div class="space-y-4">
				<div>
					<p class="mb-2 text-xs font-semibold tracking-wide text-muted-foreground uppercase">
						Basic
					</p>
					<div class="space-y-3">
						<CodeBlock
							title="Create with flags"
							code="mir device create --name my-device --namespace default --id device-001"
							lang="bash"
						/>
						<CodeBlock
							title="Shortcut  —  name/namespace positional"
							code="mir device create my-device/default --id device-001"
							lang="bash"
						/>
						<CodeBlock
							title="Random device ID"
							code="mir device create my-device/default --random-id"
							lang="bash"
						/>
					</div>
				</div>
				<div>
					<p class="mb-2 text-xs font-semibold tracking-wide text-muted-foreground uppercase">
						Labels & annotations
					</p>
					<div class="space-y-3">
						<CodeBlock
							title="With labels  —  key=value pairs separated by ;"
							code={"mir device create my-device/default --id device-001 --labels='env=prod;region=us-east'"}
							lang="bash"
						/>
						<CodeBlock
							title="With annotations"
							code={"mir device create my-device/default --id device-001 --anno='owner=team-a;ticket=MIR-42'"}
							lang="bash"
						/>
					</div>
				</div>
				<div>
					<p class="mb-2 text-xs font-semibold tracking-wide text-muted-foreground uppercase">
						Declarative
					</p>
					<div class="space-y-3">
						<CodeBlock
							title="Generate a template"
							code="mir device create -j > device.yaml"
							lang="bash"
						/>
						<CodeBlock
							title="Create from file"
							code="mir device create -f device.yaml"
							lang="bash"
						/>
						<CodeBlock
							title="Pipe from stdin"
							code="cat device.yaml | mir device create"
							lang="bash"
						/>
					</div>
				</div>
			</div>
		</Tabs.Content>
		<Tabs.Content value="gosdk">
			<CodeBlock
				title="Create a device"
				code={`package main

import (
    "fmt"
    "github.com/maxthom/mir/pkgs/module/mir"
    "github.com/maxthom/mir/pkgs/mir_v1"
)

func main() {
    m, err := mir.Connect("my-module", "nats://localhost:4222")
    if err != nil {
        panic(err)
    }
    defer m.Disconnect()

    // Create a minimal device
    device := mir_v1.NewDevice().
        WithMeta(mir_v1.Meta{Name: "my-device", Namespace: "default"}).
        WithSpec(mir_v1.DeviceSpec{DeviceId: "device-001"})

    // With labels and annotations
    deviceWithMeta := mir_v1.NewDevice().
        WithMeta(mir_v1.Meta{
            Name:        "my-device",
            Namespace:   "default",
            Labels:      map[string]string{"env": "prod"},
            Annotations: map[string]string{"owner": "team-a"},
        }).
        WithSpec(mir_v1.DeviceSpec{DeviceId: "device-001"})

    created, err := m.Client().CreateDevice().Request(deviceWithMeta)
    fmt.Println(device, created, err)
}`}
				lang="go"
			/>
		</Tabs.Content>
		<Tabs.Content value="tssdk">
			<CodeBlock
				title="Create a device"
				code={`import { Mir, Device, Meta, DeviceSpec } from '@mir/web-sdk';

async function main() {
    const mir = await Mir.connect("my-module", { servers: "ws://localhost:9222" });

    // Create a minimal device
    const device = new Device({
        meta: new Meta({ name: "my-device", namespace: "default" }),
        spec: new DeviceSpec({ deviceId: "device-001" }),
    });

    // With labels and annotations
    const deviceWithMeta = new Device({
        meta: new Meta({
            name: "my-device",
            namespace: "default",
            labels: { env: "prod" },
            annotations: { owner: "team-a" },
        }),
        spec: new DeviceSpec({ deviceId: "device-001", disabled: false }),
    });

    const created = await mir.client().createDevices().request(deviceWithMeta);
    console.log(created);

    await mir.disconnect();
}

main();`}
				lang="typescript"
			/>
		</Tabs.Content>
	</div>
</Tabs.Root>
