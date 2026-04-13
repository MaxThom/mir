<script lang="ts">
	import * as Sheet from '$lib/shared/components/shadcn/sheet/index.js';
	import * as Tabs from '$lib/shared/components/shadcn/tabs/index.js';
	import { Separator } from '$lib/shared/components/shadcn/separator/index.js';
	import CodeBlock from '$lib/shared/components/ui/code-block/code-block.svelte';
	import { docsStore } from '../stores/docs.svelte';
	import type { DocTab } from '../types/docs';
</script>

<Sheet.Header class="px-4 pt-4 pb-0">
	<Sheet.Title>Get Started</Sheet.Title>
</Sheet.Header>
<Separator class="mt-3" />
<div class="bg-muted/50 px-4 py-3 text-sm leading-relaxed text-muted-foreground">
	Welcome to Mir IoT Hub — connect devices, explore data, and build dashboards.
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
			<div class="space-y-5">
				<div>
					<p class="mb-2 text-xs font-semibold tracking-wide text-muted-foreground uppercase">
						1. Register a device
					</p>
					<p class="text-sm text-muted-foreground">
						Go to <span class="font-medium text-foreground">Devices</span> in the sidebar and click
						<span class="font-medium text-foreground">Create Device</span>. Give it a name and
						namespace — that's all you need to get started.
					</p>
				</div>
				<div>
					<p class="mb-2 text-xs font-semibold tracking-wide text-muted-foreground uppercase">
						2. Connect your device
					</p>
					<p class="text-sm text-muted-foreground">
						Integrate using the CLI, Go SDK, or TypeScript SDK (see the other tabs). Publish
						telemetry, report properties, and listen for commands from your code.
					</p>
				</div>
				<div>
					<p class="mb-2 text-xs font-semibold tracking-wide text-muted-foreground uppercase">
						3. Explore your data
					</p>
					<p class="text-sm text-muted-foreground">
						Open any device from the Devices page to view live telemetry, send commands, push
						configuration updates, and read its event history.
					</p>
				</div>
				<div>
					<p class="mb-2 text-xs font-semibold tracking-wide text-muted-foreground uppercase">
						4. Build a dashboard
					</p>
					<p class="text-sm text-muted-foreground">
						Go to <span class="font-medium text-foreground">Dashboards</span> in the sidebar. Create
						a dashboard, enter edit mode, and add widgets — charts, device lists, event feeds, text,
						and more.
					</p>
				</div>
				<div class="border-t pt-4">
					<p class="text-sm text-muted-foreground">
						Full architecture, SDK reference, and deployment guides are in the
						<a href="/book" class="font-medium text-foreground underline underline-offset-2 hover:text-foreground/80">Mir book</a>.
					</p>
				</div>
			</div>
		</Tabs.Content>
		<Tabs.Content value="cli">
			<div class="space-y-4">
				<div>
					<p class="mb-2 text-xs font-semibold tracking-wide text-muted-foreground uppercase">
						Private repository setup
					</p>
					<div class="space-y-3">
						<CodeBlock
							title="Tell Git to use SSH for the Mir repo"
							code={`# ~/.gitconfig
[url "ssh://git@github.com/maxthom/mir"]
  insteadOf = https://github.com/maxthom/mir`}
							lang="bash"
						/>
						<CodeBlock
							title="Bypass Go module proxy for Mir packages"
							code="go env -w GOPRIVATE=github.com/maxthom/mir"
							lang="bash"
						/>
					</div>
				</div>
				<div>
					<p class="mb-2 text-xs font-semibold tracking-wide text-muted-foreground uppercase">
						Install
					</p>
					<CodeBlock
						title="Install the Mir CLI"
						code="go install github.com/maxthom/mir/cmds/mir@latest"
						lang="bash"
					/>
				</div>
				<div>
					<p class="mb-2 text-xs font-semibold tracking-wide text-muted-foreground uppercase">
						Config file
					</p>
					<p class="mb-3 text-sm text-muted-foreground">
						The CLI reads <span class="font-mono text-xs text-foreground">~/.config/mir/cli.yaml</span> by default. Use <span class="font-mono text-xs text-foreground">-C</span> to point to a different file.
					</p>
					<div class="space-y-3">
						<CodeBlock
							title="View and edit the config file"
							code={`mir tools config view
mir tools config edit`}
							lang="bash"
						/>
						<CodeBlock
							title="~/.config/mir/cli.yaml"
							code={`logLevel: info
currentContext: local
contexts:
  - name: local
    target: nats://localhost:4222
    grafana: localhost:3000
    credentials: ""
  - name: prod
    target: nats://mir.prod.example.com:4222
    grafana: grafana.example.com
    credentials: ~/.config/mir/creds/cli_prod.creds`}
							lang="bash"
						/>
						<CodeBlock
							title="List and switch contexts"
							code={`mir context
mir context prod`}
							lang="bash"
						/>
					</div>
				</div>
			</div>
		</Tabs.Content>
		<Tabs.Content value="gosdk">
			<div class="space-y-4">
				<div>
					<p class="mb-2 text-xs font-semibold tracking-wide text-muted-foreground uppercase">
						Private repository setup
					</p>
					<div class="space-y-3">
						<CodeBlock
							title="Tell Git to use SSH for the Mir repo"
							code={`# ~/.gitconfig
[url "ssh://git@github.com/maxthom/mir"]
  insteadOf = https://github.com/maxthom/mir`}
							lang="bash"
						/>
						<CodeBlock
							title="Bypass Go module proxy for Mir packages"
							code="go env -w GOPRIVATE=github.com/maxthom/mir"
							lang="bash"
						/>
					</div>
				</div>
				<div>
					<p class="mb-2 text-xs font-semibold tracking-wide text-muted-foreground uppercase">
						Install
					</p>
					<CodeBlock
						title="Add the SDK to your module"
						code="go get github.com/maxthom/mir/"
						lang="bash"
					/>
				</div>
				<div>
					<p class="mb-2 text-xs font-semibold tracking-wide text-muted-foreground uppercase">
						Connect
					</p>
					<CodeBlock
						title="Connect to Mir"
						code={`package main

import (
    "fmt"
    "github.com/maxthom/mir/pkgs/module/mir"
)

func main() {
    m, err := mir.Connect("my-module", "nats://localhost:4222")
    if err != nil {
        panic(err)
    }
    defer m.Disconnect()

    fmt.Println("connected")
}`}
						lang="go"
					/>
				</div>
			</div>
		</Tabs.Content>
		<Tabs.Content value="tssdk">
			<div class="space-y-4">
				<div>
					<p class="mb-2 text-xs font-semibold tracking-wide text-muted-foreground uppercase">
						Install
					</p>
					<CodeBlock
						title="Add the SDK to your project"
						code="npm install @mir/sdk"
						lang="bash"
					/>
				</div>
				<div>
					<p class="mb-2 text-xs font-semibold tracking-wide text-muted-foreground uppercase">
						Connect
					</p>
					<CodeBlock
						title="Connect to Mir"
						code={`import { Mir } from '@mir/sdk';

async function main() {
    const mir = await Mir.connect("my-module", { servers: "ws://localhost:9222" });

    console.log("connected");
    await mir.disconnect();
}

main();`}
						lang="typescript"
					/>
				</div>
			</div>
		</Tabs.Content>
	</div>
</Tabs.Root>
