<script lang="ts">
	import * as Sheet from '$lib/shared/components/shadcn/sheet/index.js';
	import * as Tabs from '$lib/shared/components/shadcn/tabs/index.js';
	import { Separator } from '$lib/shared/components/shadcn/separator/index.js';
	import CodeBlock from '$lib/shared/components/ui/code-block/code-block.svelte';
	import { docsStore } from '../stores/docs.svelte';
	import type { DocTab } from '../types/docs';
</script>

<Sheet.Header class="px-4 pt-4 pb-0">
	<Sheet.Title>Dashboard</Sheet.Title>
</Sheet.Header>
<Separator class="mt-3" />
<div class="bg-muted/50 px-4 py-3 text-sm leading-relaxed text-muted-foreground">
	Get started with Mir — install the CLI, Go SDK, or TypeScript SDK.
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
				You're already here. Use the sidebar to navigate devices, telemetry, commands,
				configurations, and events.
			</p>
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
