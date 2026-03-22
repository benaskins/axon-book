<script lang="ts">
	import { getContext } from 'svelte';

	let { fill = '#ffd700', opacity = 0.3 } = $props();

	const { data, xGet, yGet, yScale, xScale } = getContext('LayerCake') as any;

	let pathD = $derived.by(() => {
		const d = $data;
		const xg = $xGet;
		const yg = $yGet;
		const ys = $yScale;
		if (!d || d.length === 0) return '';

		const baseline = ys(0) ?? ys.range()[0];
		const points = d.map((point: any) => `${xg(point)},${yg(point)}`).join('L');
		const lastX = xg(d[d.length - 1]);
		const firstX = xg(d[0]);
		return `M${firstX},${baseline}L${points}L${lastX},${baseline}Z`;
	});
</script>

<path d={pathD} {fill} fill-opacity={opacity} stroke="none" />
