<script lang="ts">
	import { getContext } from 'svelte';

	let { stroke = '#ffd700', strokeWidth = 2 } = $props();

	const { data, xGet, yGet } = getContext('LayerCake') as any;

	let pathD = $derived.by(() => {
		const d = $data;
		const xg = $xGet;
		const yg = $yGet;
		if (!d || d.length === 0) return '';
		return (
			'M' +
			d
				.map((point: any) => {
					return `${xg(point)},${yg(point)}`;
				})
				.join('L')
		);
	});
</script>

<path d={pathD} fill="none" {stroke} stroke-width={strokeWidth} stroke-linejoin="round" stroke-linecap="round" />
