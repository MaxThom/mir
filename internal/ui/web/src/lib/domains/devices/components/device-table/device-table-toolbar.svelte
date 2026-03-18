<script lang="ts">
	import SearchIcon from '@lucide/svelte/icons/search';
	import PlusIcon from '@lucide/svelte/icons/plus';
	import { Input } from '$lib/shared/components/shadcn/input';
	import { Badge } from '$lib/shared/components/shadcn/badge';
	import { Button } from '$lib/shared/components/shadcn/button';
	import { RefreshButtonGroup } from '$lib/shared/components/ui/refresh-button-group';
	import { ROUTES } from '$lib/shared/constants/routes';

	let {
		deviceCount,
		onlineCount,
		globalFilter,
		isLoading = false,
		onRefresh,
		onglobalfilterchange
	}: {
		deviceCount: number;
		onlineCount: number;
		globalFilter: string;
		isLoading?: boolean;
		onRefresh?: () => void;
		onglobalfilterchange: (value: string) => void;
	} = $props();
</script>

<div class="flex items-center justify-between border-b px-6 py-4">
	<div class="flex items-center gap-3">
		<span class="text-sm font-semibold">Devices</span>
		<Badge variant="secondary" class="tabular-nums">{deviceCount}</Badge>
		<div class="relative">
			<SearchIcon
				class="pointer-events-none absolute top-1/2 left-2.5 h-3.5 w-3.5 -translate-y-1/2 text-muted-foreground"
			/>
			<Input
				type="search"
				placeholder="Search…"
				class="h-7 w-48 rounded-md pl-8 text-xs transition-[width] focus:w-64"
				value={globalFilter}
				oninput={(e) => onglobalfilterchange((e.target as HTMLInputElement).value)}
			/>
		</div>
	</div>
	<div class="flex items-center gap-3">
		<div class="flex items-center gap-1.5 text-xs text-muted-foreground">
			<span class="h-1.5 w-1.5 rounded-full bg-emerald-500"></span>
			{onlineCount} online
		</div>
		<Button
			variant="ghost"
			size="icon-sm"
			class="h-7 w-7 text-emerald-600 hover:bg-emerald-50 hover:text-emerald-700 dark:text-emerald-400 dark:hover:bg-emerald-950 dark:hover:text-emerald-300"
			href={ROUTES.DEVICES.CREATE}
		>
			<PlusIcon class="h-4 w-4" />
		</Button>
		<RefreshButtonGroup {isLoading} {onRefresh} />
	</div>
</div>
