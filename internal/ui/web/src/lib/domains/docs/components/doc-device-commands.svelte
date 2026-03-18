<script lang="ts">
	import * as Sheet from '$lib/shared/components/shadcn/sheet/index.js';
	import * as Tabs from '$lib/shared/components/shadcn/tabs/index.js';
	import { Separator } from '$lib/shared/components/shadcn/separator/index.js';
	import CodeBlock from '$lib/shared/components/ui/code-block/code-block.svelte';
	import { docsStore } from '../stores/docs.svelte';
	import type { DocTab } from '../types/docs';
</script>

<Sheet.Header class="px-4 pt-4 pb-0">
	<Sheet.Title>Commands</Sheet.Title>
</Sheet.Header>
<Separator class="mt-3" />
<div class="bg-muted/50 px-4 py-3 text-sm leading-relaxed text-muted-foreground">
	Send commands to the device and inspect responses.
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
						Editor
					</p>
					<ul class="space-y-1.5 text-sm text-muted-foreground">
						<li>Selecting a command pre-fills the editor with its JSON template.</li>
						<li>
							<span class="font-medium text-foreground">Dry Run</span> validates the payload against the
							device schema without sending it.
						</li>
						<li>
							<span class="font-medium text-foreground">VIM</span> toggles vim keybindings. Use
							<kbd class="rounded border bg-muted px-1.5 py-0.5 font-mono text-[11px] text-muted-foreground">:w</kbd>
							or
							<kbd class="rounded border bg-muted px-1.5 py-0.5 font-mono text-[11px] text-muted-foreground">:wq</kbd>
							to send. Persists across tabs.
						</li>
					</ul>
				</div>
				<div>
					<p class="mb-2 text-xs font-semibold tracking-wide text-muted-foreground uppercase">
						Response
					</p>
					<p class="text-sm text-muted-foreground">
						The response panel shows a per-device status:
						<span class="font-medium text-emerald-600 dark:text-emerald-400">SUCCESS</span>,
						<span class="font-medium text-destructive">ERROR</span>,
						<span class="font-medium text-yellow-600 dark:text-yellow-400">VALIDATED</span> (dry run passed), or
						<span class="font-medium text-muted-foreground">PENDING</span>.
					</p>
				</div>
			</div>
		</Tabs.Content>
		<Tabs.Content value="cli">
			<div class="space-y-4">
				<div>
					<p class="mb-2 text-xs font-semibold tracking-wide text-muted-foreground uppercase">
						List
					</p>
					<CodeBlock
						title="List available commands"
						code="mir device cmd list my-device/default"
						lang="bash"
					/>
				</div>
				<div>
					<p class="mb-2 text-xs font-semibold tracking-wide text-muted-foreground uppercase">
						Send
					</p>
					<div class="space-y-3">
						<CodeBlock
							title="Show JSON template"
							code="mir device cmd send my-device/default -n reboot -j"
							lang="bash"
						/>
						<CodeBlock
							title="Send with inline payload"
							code={"mir device cmd send my-device/default -n reboot -p '{\"delay\":5}'"}
							lang="bash"
						/>
						<CodeBlock
							title="Interactive edit"
							code="mir device cmd send my-device/default -n reboot -e"
							lang="bash"
						/>
						<CodeBlock
							title="Dry run"
							code={"mir device cmd send my-device/default -n reboot -p '{}' --dry-run"}
							lang="bash"
						/>
						<CodeBlock
							title="Declarative — edit file then send"
							code={`mir device cmd send my-device/default -n reboot -j > payload.json
# edit payload.json
cat payload.json | mir device cmd send my-device/default -n reboot`}
							lang="bash"
						/>
					</div>
				</div>
			</div>
		</Tabs.Content>
		<Tabs.Content value="gosdk">
			<CodeBlock
				title="List and send commands"
				code={`package main

import (
    "fmt"
    "github.com/maxthom/mir/pkgs/module/mir"
    "github.com/maxthom/mir/pkgs/mir_v1"
    mir_apiv1 "github.com/maxthom/mir/pkgs/api/gen/proto/mir_api/v1"
    mypb "github.com/myorg/mydevice/proto"
)

func main() {
    m, err := mir.Connect("my-module", "nats://localhost:4222")
    if err != nil {
        panic(err)
    }
    defer m.Disconnect()

    target := mir_v1.MirDeviceTargetToProtoDeviceTarget(mir_v1.DeviceTarget{
        Names:      []string{"my-device"},
        Namespaces: []string{"default"},
    })

    // List available commands
    groups, err := m.Client().ListCommands().Request(
        &mir_apiv1.SendListCommandsRequest{Targets: target},
    )

    // Send with JSON payload
    responses, err := m.Client().SendCommand().Request(
        &mir_apiv1.SendCommandRequest{
            Targets:         target,
            Name:            "reboot",
            Payload:         []byte(\`{"delay":5}\`),
            PayloadEncoding: mir_apiv1.Encoding_ENCODING_JSON,
        },
    )

    // Send with proto message (name inferred from message type)
    responses, err = m.Client().SendCommand().RequestProto(
        &mir.SendDeviceCommandRequestProto{
            Targets: target,
            Command: &mypb.RebootCommand{Delay: 5},
        },
    )

    // Dry run
    dryResponses, err := m.Client().SendCommand().RequestProto(
        &mir.SendDeviceCommandRequestProto{
            Targets: target,
            Command: &mypb.RebootCommand{Delay: 5},
            DryRun:  true,
        },
    )

    fmt.Println(groups, responses, dryResponses, err)
}`}
				lang="go"
			/>
		</Tabs.Content>
		<Tabs.Content value="tssdk">
			<CodeBlock
				title="List and send commands"
				code={`import { Mir } from '@mir/web-sdk';

async function main() {
    const mir = await Mir.connect("my-module", { servers: "ws://localhost:9222" });

    const target = { names: ["my-device"], namespaces: ["default"] };

    // List available commands
    const groups = await mir.client().listCommands().request(target);

    // Send a command
    const responses = await mir.client().sendCommand().request(
        target,
        "reboot",
        JSON.stringify({ delay: 5 }),
        false,
    );

    // Dry run
    const dryResponses = await mir.client().sendCommand().request(
        target,
        "reboot",
        JSON.stringify({ delay: 5 }),
        true,
    );

    console.log(groups, responses, dryResponses);
    await mir.disconnect();
}

main();`}
				lang="typescript"
			/>
		</Tabs.Content>
	</div>
</Tabs.Root>
