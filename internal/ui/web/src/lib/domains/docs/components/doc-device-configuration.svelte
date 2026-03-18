<script lang="ts">
	import * as Sheet from '$lib/shared/components/shadcn/sheet/index.js';
	import * as Tabs from '$lib/shared/components/shadcn/tabs/index.js';
	import { Separator } from '$lib/shared/components/shadcn/separator/index.js';
	import CodeBlock from '$lib/shared/components/ui/code-block/code-block.svelte';
	import { docsStore } from '../stores/docs.svelte';
	import type { DocTab } from '../types/docs';
</script>

<Sheet.Header class="px-4 pt-4 pb-0">
	<Sheet.Title>Configuration</Sheet.Title>
</Sheet.Header>
<Separator class="mt-3" />
<div class="bg-muted/50 px-4 py-3 text-sm leading-relaxed text-muted-foreground">
	Send configuration payloads to devices and track acknowledgement.
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
						<li>
							<span class="font-medium text-foreground">VALUES</span> pre-fills the editor with the
							device's current configuration values.
							<span class="font-medium text-foreground">TEMPLATE</span> pre-fills with the schema
							structure and defaults.
						</li>
						<li>
							<span class="font-medium text-foreground">Dry Run</span> validates the payload against
							the device schema without sending it.
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
						<span class="font-medium text-yellow-600 dark:text-yellow-400">VALIDATED</span> (dry run
						passed),
						<span class="font-medium text-muted-foreground">NOCHANGE</span> (config already matches), or
						<span class="font-medium text-muted-foreground">PENDING</span> (device hasn't confirmed yet).
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
					<div class="space-y-3">
						<CodeBlock
							title="List available configurations"
							code="mir device cfg list my-device/default"
							lang="bash"
						/>
						<CodeBlock
							title="Show JSON template"
							code="mir device cfg list my-device/default -j"
							lang="bash"
						/>
						<CodeBlock
							title="Show current device values"
							code="mir device cfg list my-device/default -c"
							lang="bash"
						/>
					</div>
				</div>
				<div>
					<p class="mb-2 text-xs font-semibold tracking-wide text-muted-foreground uppercase">
						Send
					</p>
					<div class="space-y-3">
						<CodeBlock
							title="Send with inline payload"
							code={"mir device cfg send my-device/default -n wifi -p '{\"ssid\":\"my-net\"}'"}
							lang="bash"
						/>
						<CodeBlock
							title="Interactive edit"
							code="mir device cfg send my-device/default -n wifi -e"
							lang="bash"
						/>
						<CodeBlock
							title="Dry run"
							code={"mir device cfg send my-device/default -n wifi -p '{}' --dry-run"}
							lang="bash"
						/>
						<CodeBlock
							title="Declarative — edit file then send"
							code={`mir device cfg send my-device/default -n wifi -c > cfg.json
# edit cfg.json
cat cfg.json | mir device cfg send my-device/default -n wifi`}
							lang="bash"
						/>
					</div>
				</div>
			</div>
		</Tabs.Content>
		<Tabs.Content value="gosdk">
			<CodeBlock
				title="List and send configurations"
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

    // List available configurations
    groups, err := m.Client().ListConfig().Request(
        &mir_apiv1.SendListConfigRequest{Targets: target},
    )

    // Send with JSON struct
    responses, err := m.Client().SendConfig().RequestJson(
        &mir.SendDeviceConfigRequestJson{
            Targets:        target,
            CommandName:    "wifi",
            CommandPayload: map[string]any{"ssid": "my-net"},
        },
    )

    // Send with proto message (name inferred from message type)
    responses, err = m.Client().SendConfig().RequestProto(
        &mir.SendDeviceConfigRequestProto{
            Targets: target,
            Command: &mypb.WifiConfig{Ssid: "my-net"},
        },
    )

    // Dry run
    dryResponses, err := m.Client().SendConfig().RequestProto(
        &mir.SendDeviceConfigRequestProto{
            Targets: target,
            Command: &mypb.WifiConfig{Ssid: "my-net"},
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
				title="List and send configurations"
				code={`import { Mir } from '@mir/web-sdk';

async function main() {
    const mir = await Mir.connect("my-module", { servers: "ws://localhost:9222" });

    const target = { names: ["my-device"], namespaces: ["default"] };

    // List available configurations
    const groups = await mir.client().listConfigs().request(target);

    // Send a configuration
    const responses = await mir.client().sendConfig().request(
        target,
        "wifi",
        JSON.stringify({ ssid: "my-net" }),
        false,
    );

    // Dry run
    const dryResponses = await mir.client().sendConfig().request(
        target,
        "wifi",
        JSON.stringify({ ssid: "my-net" }),
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
