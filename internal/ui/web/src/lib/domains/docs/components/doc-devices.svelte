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
			<div class="space-y-4">
				<div>
					<p class="mb-2 text-xs font-semibold tracking-wide text-muted-foreground uppercase">
						Search
					</p>
					<p class="text-sm text-muted-foreground">
						Filters across all columns — name, namespace, device ID, labels, status, schema packages, and last heartbeat.
					</p>
				</div>
				<div>
					<p class="mb-2 text-xs font-semibold tracking-wide text-muted-foreground uppercase">
						Multi-select
					</p>
					<p class="text-sm text-muted-foreground">
						Select devices using the row checkboxes to reveal a bulk action bar — send telemetry, commands, or configuration to all selected devices at once.
					</p>
				</div>
				<div>
					<p class="mb-2 text-xs font-semibold tracking-wide text-muted-foreground uppercase">
						Keyboard shortcuts
					</p>
					<div class="space-y-1.5">
						<div class="flex items-center justify-between text-sm">
							<span class="text-muted-foreground">Select all</span>
							<div class="flex gap-1">
								<kbd class="rounded border bg-muted px-1.5 py-0.5 font-mono text-[11px] text-muted-foreground">Ctrl</kbd>
								<kbd class="rounded border bg-muted px-1.5 py-0.5 font-mono text-[11px] text-muted-foreground">A</kbd>
							</div>
						</div>
						<div class="flex items-center justify-between text-sm">
							<span class="text-muted-foreground">Clear selection</span>
							<kbd class="rounded border bg-muted px-1.5 py-0.5 font-mono text-[11px] text-muted-foreground">Esc</kbd>
						</div>
						<div class="flex items-center justify-between text-sm">
							<span class="text-muted-foreground">Bulk telemetry</span>
							<kbd class="rounded border bg-muted px-1.5 py-0.5 font-mono text-[11px] text-muted-foreground">T</kbd>
						</div>
						<div class="flex items-center justify-between text-sm">
							<span class="text-muted-foreground">Bulk commands</span>
							<kbd class="rounded border bg-muted px-1.5 py-0.5 font-mono text-[11px] text-muted-foreground">C</kbd>
						</div>
						<div class="flex items-center justify-between text-sm">
							<span class="text-muted-foreground">Bulk configuration</span>
							<kbd class="rounded border bg-muted px-1.5 py-0.5 font-mono text-[11px] text-muted-foreground">P</kbd>
						</div>
					</div>
					<p class="mt-2 text-xs text-muted-foreground">T, C, P only active with a selection.</p>
				</div>
			</div>
		</Tabs.Content>
		<Tabs.Content value="cli">
			<div class="space-y-4">
				<div>
					<p class="mb-2 text-xs font-semibold tracking-wide text-muted-foreground uppercase">
						Basic listing
					</p>
					<div class="space-y-3">
						<CodeBlock title="List all devices" code="mir device list" lang="bash" />
						<CodeBlock
							title="List a single device by name"
							code="mir device list my-device"
							lang="bash"
						/>
						<CodeBlock
							title="List all devices in a namespace"
							code="mir device list /default"
							lang="bash"
						/>
						<CodeBlock
							title="List a single device by name and namespace"
							code="mir device list my-device/default"
							lang="bash"
						/>
					</div>
				</div>
				<div>
					<p class="mb-2 text-xs font-semibold tracking-wide text-muted-foreground uppercase">
						Output format
					</p>
					<p class="mb-2 text-xs text-muted-foreground">
						Use <code class="font-mono">-o</code> to choose between
						<code class="font-mono">pretty</code> (default), <code class="font-mono">json</code>, or
						<code class="font-mono">yaml</code>.
					</p>
					<div class="space-y-3">
						<CodeBlock
							title="JSON output"
							code="mir device list -o json"
							lang="bash"
						/>
						<CodeBlock
							title="YAML output  —  pipe into a file"
							code="mir device list my-device -o yaml > device.yaml"
							lang="bash"
						/>
					</div>
				</div>
				<div>
					<p class="mb-2 text-xs font-semibold tracking-wide text-muted-foreground uppercase">
						Targeting
					</p>
					<p class="mb-2 text-xs text-muted-foreground">
						Use target flags to filter devices. Flags accept comma-separated values.
					</p>
					<div class="space-y-3">
						<CodeBlock
							title="Filter by IDs"
							code="mir device list --target.ids=abc123,def456"
							lang="bash"
						/>
						<CodeBlock
							title="Filter by names"
							code="mir device list --target.names=sensor-1,sensor-2"
							lang="bash"
						/>
						<CodeBlock
							title="All devices in a namespace"
							code="mir device list --target.namespaces=production"
							lang="bash"
						/>
						<CodeBlock
							title="Filter by labels  —  key=value pairs separated by ;"
							code={"mir device list --target.labels='env=prod;region=us-east'"}
							lang="bash"
						/>
					</div>
				</div>
				<div>
					<p class="mb-2 text-xs font-semibold tracking-wide text-muted-foreground uppercase">
						Performance
					</p>
					<p class="mb-2 text-xs text-muted-foreground">
						Skip event history to speed up the query on large fleets.
					</p>
					<CodeBlock
						title="Exclude events"
						code="mir device list --exclude-events"
						lang="bash"
					/>
				</div>
			</div>
		</Tabs.Content>
		<Tabs.Content value="gosdk">
			<CodeBlock
				title="List devices"
				code={`package main

import (
    "fmt"
    mir "github.com/maxthom/mir/pkgs/module/mir"
    "github.com/maxthom/mir/pkgs/mir_v1"
)

func main() {
    m, err := mir.Connect("my-module", "nats://localhost:4222")
    if err != nil {
        panic(err)
    }
    defer m.Disconnect()

    // List all devices
    devices, err := m.Client().ListDevice().Request(mir_v1.DeviceTarget{}, false)

    // List by name
    devices, err = m.Client().ListDevice().Request(
        mir_v1.DeviceTarget{Names: []string{"my-device"}},
        false,
    )

    // List all in a namespace
    devices, err = m.Client().ListDevice().Request(
        mir_v1.DeviceTarget{Namespaces: []string{"default"}},
        false,
    )

    // List by name and namespace
    devices, err = m.Client().ListDevice().Request(
        mir_v1.DeviceTarget{Names: []string{"my-device"}, Namespaces: []string{"default"}},
        false,
    )

    fmt.Println(devices, err)
}`}
				lang="go"
			/>
		</Tabs.Content>
		<Tabs.Content value="tssdk">
			<CodeBlock
				title="List devices"
				code={`import { Mir } from '@mir/web-sdk';

async function main() {
    const mir = await Mir.connect("my-module", { servers: "ws://localhost:9222" });

    // List all devices
    const all = await mir.client().listDevices().request({}, false);

    // List by name
    const byName = await mir.client().listDevices().request(
        { names: ["my-device"] },
        false
    );

    // List all in a namespace
    const byNs = await mir.client().listDevices().request(
        { namespaces: ["default"] },
        false
    );

    // List by name and namespace
    const byBoth = await mir.client().listDevices().request(
        { names: ["my-device"], namespaces: ["default"] },
        false
    );

    await mir.disconnect();
}

main();`}
				lang="typescript"
			/>
		</Tabs.Content>
	</div>
</Tabs.Root>
