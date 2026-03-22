<script lang="ts">
	import { getContext } from 'svelte';

	interface Series {
		key: string;
		color: string;
	}

	let { series = [] as Series[] } = $props();

	const { data, xGet, yScale } = getContext('LayerCake') as any;

	let stacks = $derived.by(() => {
		const d = $data;
		const xg = $xGet;
		const ys = $yScale;
		if (!d || d.length === 0 || series.length === 0) return [];

		const baseline = ys(0) ?? ys.range()[0];
		const results: { color: string; path: string }[] = [];

		// Compute cumulative values for stacking
		const cumulative = d.map(() => 0);

		for (const s of series) {
			const topPoints: string[] = [];
			const bottomPoints: string[] = [];

			for (let i = 0; i < d.length; i++) {
				const x = xg(d[i]);
				const val = parseFloat(d[i][s.key]) || 0;
				const bottom = cumulative[i];
				const top = bottom + val;
				cumulative[i] = top;

				topPoints.push(`${x},${ys(top)}`);
				bottomPoints.unshift(`${x},${ys(bottom)}`);
			}

			const path = `M${topPoints.join('L')}L${bottomPoints.join('L')}Z`;
			results.push({ color: s.color, path });
		}

		return results;
	});
</script>

{#each stacks as stack}
	<path d={stack.path} fill={stack.color} fill-opacity="0.7" stroke={stack.color} stroke-width="0.5" />
{/each}
