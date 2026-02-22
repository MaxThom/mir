<script lang="ts">
	import { Skeleton } from '$lib/shared/components/shadcn/skeleton';
	import * as Table from '$lib/shared/components/shadcn/table';
	import { deviceColumns } from './device-columns';
</script>

<Table.Root class="min-w-225">
	<Table.Header>
		<Table.Row class="hover:bg-transparent">
			{#each deviceColumns as col, i (i)}
				<Table.Head class="h-10 text-xs font-medium tracking-wide text-muted-foreground uppercase">
					{col.header as string}
				</Table.Head>
			{/each}
		</Table.Row>
	</Table.Header>
	<Table.Body>
		{#each { length: 5 }, i (i)}
			<Table.Row class="hover:bg-transparent">
				{#each deviceColumns, j (j)}
					<Table.Cell><Skeleton class="h-4 w-full" /></Table.Cell>
				{/each}
			</Table.Row>
		{/each}
	</Table.Body>
</Table.Root>
