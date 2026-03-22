<script lang="ts">
	import { getContext } from 'svelte';

	let { colorFn = (_d: any) => '#ffd700', strokeWidth = 2 } = $props();

	const { data, xGet, yGet } = getContext('LayerCake') as any;

	let segments = $derived.by(() => {
		const d = $data;
		const xg = $xGet;
		const yg = $yGet;
		if (!d || d.length < 2) return [];

		const result: { path: string; color: string }[] = [];
		for (let i = 0; i < d.length - 1; i++) {
			const x1 = xg(d[i]);
			const y1 = yg(d[i]);
			const x2 = xg(d[i + 1]);
			const y2 = yg(d[i + 1]);
			result.push({
				path: `M${x1},${y1}L${x2},${y2}`,
				color: colorFn(d[i])
			});
		}
		return result;
	});
</script>

{#each segments as seg}
	<path d={seg.path} fill="none" stroke={seg.color} stroke-width={strokeWidth} stroke-linecap="round" />
{/each}
