<script lang="ts">
	import * as Sheet from '$lib/shared/components/shadcn/sheet/index.js';
	import { Separator } from '$lib/shared/components/shadcn/separator/index.js';
</script>

<Sheet.Header class="px-4 pt-4 pb-0">
	<Sheet.Title>Dashboards</Sheet.Title>
</Sheet.Header>
<Separator class="mt-3" />
<div class="bg-muted/50 px-4 py-3 text-sm leading-relaxed text-muted-foreground">
	Build and manage custom dashboards from widgets — charts, device lists, event feeds, and more.
</div>
<Separator />
<div class="flex-1 overflow-y-auto px-4 py-4">
	<div class="space-y-5">
		<div>
			<p class="mb-2 text-xs font-semibold tracking-wide text-muted-foreground uppercase">
				Overview
			</p>
			<p class="text-sm text-muted-foreground">
				Each dashboard belongs to a namespace and holds a grid of widgets. Pin dashboards to the tab
				bar at the top to switch between them quickly.
			</p>
		</div>
		<div>
			<p class="mb-2 text-xs font-semibold tracking-wide text-muted-foreground uppercase">
				Create a dashboard
			</p>
			<ol class="space-y-1.5 text-sm text-muted-foreground">
				<li class="flex gap-2">
					<span class="shrink-0 font-mono text-xs text-foreground">1.</span>
					Click <span class="mx-0.5 rounded border bg-muted px-1.5 py-0.5 font-mono text-[11px] text-foreground">⋯</span> in the top-right corner.
				</li>
				<li class="flex gap-2">
					<span class="shrink-0 font-mono text-xs text-foreground">2.</span>
					Select <span class="font-medium text-foreground">New Dashboard</span>.
				</li>
				<li class="flex gap-2">
					<span class="shrink-0 font-mono text-xs text-foreground">3.</span>
					Enter a namespace and a name, then confirm with
					<span class="mx-0.5 rounded border bg-muted px-1.5 py-0.5 font-mono text-[11px] text-foreground">✓</span>.
				</li>
			</ol>
		</div>
		<div>
			<p class="mb-2 text-xs font-semibold tracking-wide text-muted-foreground uppercase">
				Add widgets
			</p>
			<ol class="space-y-1.5 text-sm text-muted-foreground">
				<li class="flex gap-2">
					<span class="shrink-0 font-mono text-xs text-foreground">1.</span>
					Click <span class="mx-0.5 rounded border bg-muted px-1.5 py-0.5 font-mono text-[11px] text-foreground">⋯</span> → <span class="font-medium text-foreground">Edit Dashboard</span> to enter edit mode.
				</li>
				<li class="flex gap-2">
					<span class="shrink-0 font-mono text-xs text-foreground">2.</span>
					Click <span class="font-medium text-foreground">Add Widget</span>, choose a widget type, and configure it.
				</li>
				<li class="flex gap-2">
					<span class="shrink-0 font-mono text-xs text-foreground">3.</span>
					Drag widgets to rearrange and resize them. Save with
					<span class="mx-0.5 rounded border bg-muted px-1.5 py-0.5 font-mono text-[11px] text-foreground">✓</span>.
				</li>
			</ol>
		</div>
		<div>
			<p class="mb-2 text-xs font-semibold tracking-wide text-muted-foreground uppercase">
				Manage dashboards
			</p>
			<div class="space-y-2 text-sm text-muted-foreground">
				<p>
					Use the <span class="mx-0.5 rounded border bg-muted px-1.5 py-0.5 font-mono text-[11px] text-foreground">⊞</span> icon dropdown to pin or unpin dashboards from the tab bar.
				</p>
				<p>
					Rename or delete a dashboard via <span class="mx-0.5 rounded border bg-muted px-1.5 py-0.5 font-mono text-[11px] text-foreground">⋯</span> → <span class="font-medium text-foreground">Edit Dashboard</span>.
				</p>
			</div>
		</div>
	</div>
</div>
