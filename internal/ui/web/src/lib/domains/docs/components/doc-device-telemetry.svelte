<script lang="ts">
	import * as Sheet from '$lib/shared/components/shadcn/sheet/index.js';
	import * as Tabs from '$lib/shared/components/shadcn/tabs/index.js';
	import { Separator } from '$lib/shared/components/shadcn/separator/index.js';
	import CodeBlock from '$lib/shared/components/ui/code-block/code-block.svelte';
	import { docsStore } from '../stores/docs.svelte';
	import type { DocTab } from '../types/docs';
</script>

<Sheet.Header class="px-4 pt-4 pb-0">
	<Sheet.Title>Telemetry</Sheet.Title>
</Sheet.Header>
<Separator class="mt-3" />
<div class="bg-muted/50 px-4 py-3 text-sm leading-relaxed text-muted-foreground">
	Query and visualize time-series data from the device.
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
						Fields
					</p>
					<p class="text-sm text-muted-foreground">
						Click a field button to display it. <span class="font-medium text-foreground">Shift-click</span> to add or remove a field without deselecting the others.
					</p>
				</div>
				<div>
					<p class="mb-2 text-xs font-semibold tracking-wide text-muted-foreground uppercase">
						Time range
					</p>
					<ul class="space-y-1.5 text-sm text-muted-foreground">
						<li>Use the <span class="font-medium text-foreground">time picker</span> for relative presets (1m – 90d) or a custom date range with start/end times.</li>
						<li><span class="font-medium text-foreground">Drag on the chart</span> to zoom into a selection. The reset button restores the previous range.</li>
					</ul>
				</div>
				<div>
					<p class="mb-2 text-xs font-semibold tracking-wide text-muted-foreground uppercase">
						Split view
					</p>
					<p class="text-sm text-muted-foreground">
						The grid icon cycles through 1, 2, 3, and 4 independent chart panels. Each slot has its own field selector.
					</p>
				</div>
				<div>
					<p class="mb-2 text-xs font-semibold tracking-wide text-muted-foreground uppercase">
						Other
					</p>
					<ul class="space-y-1.5 text-sm text-muted-foreground">
						<li>The <span class="font-medium text-foreground">copy icon</span> exports the current query result as CSV.</li>
						<li>The <span class="font-medium text-foreground">external link icon</span> opens the measurement in Grafana Explore (requires a Grafana context).</li>
						<li><kbd class="rounded border bg-muted px-1.5 py-0.5 font-mono text-[11px] text-muted-foreground">Esc</kbd> exits fullscreen.</li>
					</ul>
				</div>
			</div>
		</Tabs.Content>
		<Tabs.Content value="cli">
			<div class="space-y-4">
				<div>
					<p class="mb-2 text-xs font-semibold tracking-wide text-muted-foreground uppercase">
						List
					</p>
					<div class="space-y-3">
						<CodeBlock
							title="List available measurements"
							code="mir device tlm list my-device/default"
							lang="bash"
						/>
						<CodeBlock
							title="Show fields and print Influx query"
							code="mir device tlm list my-device/default -s --print-query"
							lang="bash"
						/>
						<CodeBlock
							title="Refresh schema if measurements are missing"
							code="mir device tlm list my-device/default -r"
							lang="bash"
						/>
					</div>
				</div>
				<div>
					<p class="mb-2 text-xs font-semibold tracking-wide text-muted-foreground uppercase">
						Query
					</p>
					<div class="space-y-3">
						<CodeBlock
							title="Query last 5 minutes"
							code="mir device tlm query my-device/default -m temperature --since 5m"
							lang="bash"
						/>
						<CodeBlock
							title="Query specific fields"
							code="mir device tlm query my-device/default -m temperature -f value,unit --since 1h"
							lang="bash"
						/>
						<CodeBlock
							title="Query absolute range"
							code="mir device tlm query my-device/default -m temperature --start 2025-01-01T00:00:00Z --end 2025-01-02T00:00:00Z"
							lang="bash"
						/>
						<CodeBlock
							title="Export as CSV"
							code="mir device tlm query my-device/default -m temperature --since 1h -o csv"
							lang="bash"
						/>
					</div>
				</div>
			</div>
		</Tabs.Content>
		<Tabs.Content value="gosdk">
			<CodeBlock
				title="List and query telemetry"
				code={`package main

import (
    "fmt"
    "time"
    "github.com/maxthom/mir/pkgs/module/mir"
    "github.com/maxthom/mir/pkgs/mir_v1"
    mir_apiv1 "github.com/maxthom/mir/pkgs/api/gen/proto/mir_api/v1"
    "google.golang.org/protobuf/types/known/timestamppb"
)

func main() {
    m, err := mir.Connect("my-module", "nats://localhost:4222")
    if err != nil {
        panic(err)
    }
    defer m.Disconnect()

    target := mir_v1.DeviceTarget{
        Names:      []string{"my-device"},
        Namespaces: []string{"default"},
    }

    // List available measurements
    groups, err := m.Client().ListTelemetry().Request(
        &mir_apiv1.ListTelemetryRequest{
            Targets: mir_v1.MirDeviceTargetToProtoDeviceTarget(target),
        },
    )

    // Query a measurement
    end := time.Now()
    start := end.Add(-5 * time.Minute)
    data, err := m.Client().QueryTelemetry().Request(
        &mir_apiv1.QueryTelemetryRequest{
            Targets:     mir_v1.MirDeviceTargetToProtoDeviceTarget(target),
            Measurement: "temperature",
            Fields:      []string{"value"},
            StartTime:   timestamppb.New(start),
            EndTime:     timestamppb.New(end),
        },
    )

    fmt.Println(groups, data, err)
}`}
				lang="go"
			/>
		</Tabs.Content>
		<Tabs.Content value="tssdk">
			<CodeBlock
				title="List and query telemetry"
				code={`import { Mir } from '@mir/web-sdk';

async function main() {
    const mir = await Mir.connect("my-module", { servers: "ws://localhost:9222" });

    const target = { names: ["my-device"], namespaces: ["default"] };

    // List available measurements
    const groups = await mir.client().listTelemetry().request(target);

    // Query a measurement
    const end = new Date();
    const start = new Date(end.getTime() - 5 * 60 * 1000);
    const data = await mir.client().queryTelemetry().request(
        target,
        "temperature",
        ["value"],
        start,
        end,
    );

    console.log(groups, data);
    await mir.disconnect();
}

main();`}
				lang="typescript"
			/>
		</Tabs.Content>
	</div>
</Tabs.Root>
