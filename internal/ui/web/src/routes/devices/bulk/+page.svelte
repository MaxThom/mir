<script lang="ts">
	import { goto } from '$app/navigation';
	import { ROUTES } from '$lib/shared/constants/routes';
	import { resolve } from '$app/paths';

	goto(resolve(ROUTES.DEVICES.LIST));
</script>
