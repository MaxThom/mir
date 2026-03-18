<script lang="ts">
	import * as Sheet from '$lib/shared/components/shadcn/sheet/index.js';
	import * as Tabs from '$lib/shared/components/shadcn/tabs/index.js';
	import { Separator } from '$lib/shared/components/shadcn/separator/index.js';
	import CodeBlock from '$lib/shared/components/ui/code-block/code-block.svelte';
	import { docsStore } from '../stores/docs.svelte';
	import type { DocTab } from '../types/docs';
</script>

<Sheet.Header class="px-4 pt-4 pb-0">
	<Sheet.Title>Events</Sheet.Title>
</Sheet.Header>
<Separator class="mt-3" />
<div class="bg-muted/50 px-4 py-3 text-sm leading-relaxed text-muted-foreground">
	Browse and filter events emitted by devices.
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
						Filters
					</p>
					<ul class="space-y-1.5 text-sm text-muted-foreground">
						<li>
							The search bar filters across all columns. The
							<span class="font-medium text-foreground">Reason</span> dropdown filters by specific
							reason values present in the current result set.
						</li>
						<li>
							<span class="font-medium text-foreground">Normal / Warning</span> toggles filter by event
							type.
						</li>
						<li>
							The calendar picker filters by date range. Click the
							<span class="font-medium text-foreground">×</span> to reset it.
						</li>
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
							title="List all events"
							code="mir event list"
							lang="bash"
						/>
						<CodeBlock
							title="Filter by device"
							code="mir event list my-device/default"
							lang="bash"
						/>
						<CodeBlock
							title="Limit results"
							code="mir event list --limit 50"
							lang="bash"
						/>
						<CodeBlock
							title="Filter by date range"
							code="mir event list --from 2025-01-01T00:00:00Z --to 2025-01-02T00:00:00Z"
							lang="bash"
						/>
					</div>
				</div>
				<div>
					<p class="mb-2 text-xs font-semibold tracking-wide text-muted-foreground uppercase">
						Delete
					</p>
					<CodeBlock
						title="Delete events for a device"
						code="mir event delete my-device/default"
						lang="bash"
					/>
				</div>
			</div>
		</Tabs.Content>
		<Tabs.Content value="gosdk">
			<CodeBlock
				title="List and delete events"
				code={`package main

import (
    "fmt"
    "time"
    "github.com/maxthom/mir/pkgs/module/mir"
    "github.com/maxthom/mir/pkgs/mir_v1"
)

func main() {
    m, err := mir.Connect("my-module", "nats://localhost:4222")
    if err != nil {
        panic(err)
    }
    defer m.Disconnect()

    // List all events for a device
    events, err := m.Client().ListEvents().Request(
        mir_v1.EventTarget{
            ObjectTarget: mir_v1.ObjectTarget{
                Names:      []string{"my-device"},
                Namespaces: []string{"default"},
            },
        },
    )

    // List with date range and limit
    end := time.Now()
    start := end.Add(-24 * time.Hour)
    filtered, err := m.Client().ListEvents().Request(
        mir_v1.EventTarget{
            ObjectTarget: mir_v1.ObjectTarget{
                Names:      []string{"my-device"},
                Namespaces: []string{"default"},
            },
            DateFilter: mir_v1.DateFilter{From: start, To: end},
            Limit:      50,
        },
    )

    // Delete events for a device
    deleted, err := m.Client().DeleteEvents().Request(
        mir_v1.EventTarget{
            ObjectTarget: mir_v1.ObjectTarget{
                Names:      []string{"my-device"},
                Namespaces: []string{"default"},
            },
        },
    )

    fmt.Println(events, filtered, deleted, err)
}`}
				lang="go"
			/>
		</Tabs.Content>
		<Tabs.Content value="tssdk">
			<CodeBlock
				title="List events"
				code={`import { Mir, EventTarget, DateFilter } from '@mir/web-sdk';

async function main() {
    const mir = await Mir.connect("my-module", { servers: "ws://localhost:9222" });

    // List all events for a device
    const events = await mir.client().listEvents().request(
        new EventTarget({
            names: ["my-device"],
            namespaces: ["default"],
        }),
    );

    // List with date range and limit
    const end = new Date();
    const start = new Date(end.getTime() - 24 * 60 * 60 * 1000);
    const filtered = await mir.client().listEvents().request(
        new EventTarget({
            names: ["my-device"],
            namespaces: ["default"],
            dateFilter: new DateFilter({ from: start, to: end }),
            limit: 50,
        }),
    );

    console.log(events, filtered);
    await mir.disconnect();
}

main();`}
				lang="typescript"
			/>
		</Tabs.Content>
	</div>
</Tabs.Root>
