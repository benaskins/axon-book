<script lang="ts">
	import { getContext } from 'svelte';

	let { radius = 3, fill = '#ef4444', opacity = 0.8 } = $props();

	const { data, xGet, yGet } = getContext('LayerCake') as any;
</script>

{#each $data as d}
	<circle cx={$xGet(d)} cy={$yGet(d)} r={radius} {fill} fill-opacity={opacity} />
{/each}
