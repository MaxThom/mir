<script lang="ts">
	import * as Sheet from '$lib/shared/components/shadcn/sheet/index.js';
	import * as Tabs from '$lib/shared/components/shadcn/tabs/index.js';
	import { Separator } from '$lib/shared/components/shadcn/separator/index.js';
	import CodeBlock from '$lib/shared/components/ui/code-block/code-block.svelte';
	import { docsStore } from '../stores/docs.svelte';
	import type { DocTab } from '../types/docs';
</script>

<Sheet.Header class="px-4 pt-4 pb-0">
	<Sheet.Title>Device overview</Sheet.Title>
</Sheet.Header>
<Separator class="mt-3" />
<div class="bg-muted/50 px-4 py-3 text-sm leading-relaxed text-muted-foreground">
	View and edit a device's metadata, spec, properties, and recent events.
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
						Editing
					</p>
					<ul class="space-y-1.5 text-sm text-muted-foreground">
						<li>
							The <span class="font-medium text-foreground">pencil icon</span> opens an inline form to
							edit name, namespace, labels, annotations, and disabled state.
						</li>
						<li>
							The <span class="font-medium text-foreground">file icon</span> opens a full YAML/JSON editor
							pre-filled with the entire device — including properties and status.
						</li>
						<li>
							<span class="font-medium text-foreground">Device ID</span> is always read-only and cannot
							be changed after creation.
						</li>
					</ul>
				</div>
				<div>
					<p class="mb-2 text-xs font-semibold tracking-wide text-muted-foreground uppercase">
						Properties
					</p>
					<ul class="space-y-1.5 text-sm text-muted-foreground">
						<li>
							<span class="font-medium text-foreground">Desired</span> are values the server wants the
							device to apply. <span class="font-medium text-foreground">Reported</span> are values the
							device has confirmed.
						</li>
					</ul>
				</div>
				<div>
					<p class="mb-2 text-xs font-semibold tracking-wide text-muted-foreground uppercase">
						Events
					</p>
					<ul class="space-y-1.5 text-sm text-muted-foreground">
						<li>Click any event row to expand its JSON payload.</li>
					</ul>
				</div>
				<div>
					<p class="mb-2 text-xs font-semibold tracking-wide text-muted-foreground uppercase">
						Delete
					</p>
					<p class="text-sm text-muted-foreground">
						Deletion requires typing <span class="font-mono font-medium text-foreground">name/namespace</span> to confirm.
					</p>
				</div>
			</div>
		</Tabs.Content>
		<Tabs.Content value="cli">
			<div class="space-y-4">
				<div>
					<p class="mb-2 text-xs font-semibold tracking-wide text-muted-foreground uppercase">
						Inspect
					</p>
					<div class="space-y-3">
						<CodeBlock
							title="Get device as YAML"
							code="mir device list my-device/default -o yaml"
							lang="bash"
						/>
						<CodeBlock
							title="Get device as JSON"
							code="mir device list my-device/default -o json"
							lang="bash"
						/>
					</div>
				</div>
				<div>
					<p class="mb-2 text-xs font-semibold tracking-wide text-muted-foreground uppercase">
						Update
					</p>
					<div class="space-y-3">
						<CodeBlock
							title="Interactive editor"
							code="mir device edit my-device/default"
							lang="bash"
						/>
						<CodeBlock
							title="Update with flags"
							code={"mir device update my-device/default --labels='env=prod' --disabled"}
							lang="bash"
						/>
						<CodeBlock
							title="Declarative  —  edit YAML then apply"
							code={`mir device list my-device/default -o yaml > device.yaml
# edit device.yaml
mir device apply -f device.yaml`}
							lang="bash"
						/>
					</div>
				</div>
				<div>
					<p class="mb-2 text-xs font-semibold tracking-wide text-muted-foreground uppercase">
						Delete
					</p>
					<CodeBlock
						title="Delete a device"
						code="mir device delete my-device/default"
						lang="bash"
					/>
				</div>
			</div>
		</Tabs.Content>
		<Tabs.Content value="gosdk">
			<CodeBlock
				title="Inspect, update and delete"
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

    // Get a single device
    devices, err := m.Client().ListDevice().Request(
        mir_v1.DeviceTarget{Names: []string{"my-device"}, Namespaces: []string{"default"}},
        false,
    )
    device := devices[0]

    // Update — modify then call RequestSingle
    device.Meta.Labels = map[string]string{"env": "prod"}
    updated, err := m.Client().UpdateDevice().RequestSingle(device)

    // Delete
    deleted, err := m.Client().DeleteDevice().Request(device.ToTarget())

    fmt.Println(updated, deleted, err)
}`}
				lang="go"
			/>
		</Tabs.Content>
		<Tabs.Content value="tssdk">
			<CodeBlock
				title="Inspect, update and delete"
				code={`import { Mir, Device, Meta, DeviceSpec } from '@mir/web-sdk';

async function main() {
    const mir = await Mir.connect("my-module", { servers: "ws://localhost:9222" });

    // Get a single device
    const [device] = await mir.client().listDevices().request(
        { names: ["my-device"], namespaces: ["default"] },
        false
    );

    // Update — modify then send
    const updated = new Device({
        ...device,
        meta: new Meta({ ...device.meta, labels: { env: "prod" } }),
        spec: new DeviceSpec({ ...device.spec, disabled: false }),
    });
    await mir.client().updateDevices().request(
        { names: ["my-device"], namespaces: ["default"] },
        updated
    );

    // Delete
    await mir.client().deleteDevices().request(
        { names: ["my-device"], namespaces: ["default"] }
    );

    await mir.disconnect();
}

main();`}
				lang="typescript"
			/>
		</Tabs.Content>
	</div>
</Tabs.Root>
