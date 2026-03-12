<script lang="ts">
	import type { QueryData } from '@mir/sdk';
	import * as Tabs from '$lib/shared/components/shadcn/tabs';
	import * as Table from '$lib/shared/components/shadcn/table';
	import CodeBlock from '$lib/shared/components/ui/code-block/code-block.svelte';
	import { editorPrefs } from '$lib/shared/stores/editor-prefs.svelte';

	let {
		data,
		exploreQuery = ''
	}: {
		data: QueryData | null;
		exploreQuery?: string;
	} = $props();

	let activeTab = $state('');

	function formatCell(value: number | boolean | string | Date | null | undefined): string {
		if (value === null || value === undefined) return '—';
		if (value instanceof Date) {
			return editorPrefs.utc
				? value.toISOString().replace('T', ' ').slice(0, 19) + ' UTC'
				: value.toLocaleString();
		}
		if (typeof value === 'boolean') return value ? 'true' : 'false';
		return String(value);
	}
</script>

{#if data || exploreQuery}
	<div class="flex h-104 flex-none flex-col border-t">
		<Tabs.Root bind:value={activeTab} activationMode="manual" class="flex h-full flex-col">
			<div class="border-b">
				<Tabs.List class="h-9 flex-none justify-start gap-0 rounded-none bg-transparent px-3">
					<Tabs.Trigger
						value="data"
						class="rounded-none text-xs"
						onclick={(e) => {
							if (activeTab === 'data') {
								e.preventDefault();
								activeTab = '';
							}
						}}>Table</Tabs.Trigger
					>
					<Tabs.Trigger
						value="query"
						class="rounded-none text-xs"
						onclick={(e) => {
							if (activeTab === 'query') {
								e.preventDefault();
								activeTab = '';
							}
						}}>Flux Query</Tabs.Trigger
					>
				</Tabs.List>
			</div>

			<Tabs.Content value="data" class="mt-0 min-h-0 flex-1 overflow-auto">
				{#if data && data.rows.length}
					<Table.Root>
						<Table.Header>
							<Table.Row>
								{#each data.headers as header, i (i)}
									<Table.Head class="h-7 px-3 text-xs whitespace-nowrap">{header}</Table.Head>
								{/each}
							</Table.Row>
						</Table.Header>
						<Table.Body>
							{#each data.rows as row, i (i)}
								<Table.Row>
									{#each data.headers as header, j (j)}
										<Table.Cell class="px-3 py-1 font-mono text-xs">
											{formatCell(row.values[header])}
										</Table.Cell>
									{/each}
								</Table.Row>
							{/each}
						</Table.Body>
					</Table.Root>
				{:else if data}
					<div class="flex h-full items-center justify-center">
						<p class="rounded-md bg-background/80 px-3 py-1.5 text-sm text-muted-foreground">
							No data in this time range.
						</p>
					</div>
				{/if}
			</Tabs.Content>

			<Tabs.Content value="query" class="mt-0 min-h-0 flex-1 overflow-auto p-3">
				{#if exploreQuery}
					<CodeBlock code={exploreQuery} lang="bash" />
				{/if}
			</Tabs.Content>
		</Tabs.Root>
	</div>
{/if}
