<script lang="ts">
	import { getContext } from 'svelte';

	let { tickCount = 5, formatTick = (d: any) => String(d) } = $props();

	const { yScale, xRange } = getContext('LayerCake') as any;

	let ticks = $derived.by(() => {
		const scale = $yScale;
		if (!scale || !scale.ticks) return [];
		return scale.ticks(tickCount);
	});

	let xLeft = $derived.by(() => {
		const range = $xRange;
		return range ? range[0] : 0;
	});
</script>

<g class="axis-y">
	{#each ticks as tick}
		<g transform="translate({xLeft}, {$yScale(tick)})">
			<line x1="0" x2="-6" stroke="#444" />
			<text x="-10" text-anchor="end" dominant-baseline="middle" fill="#888" font-size="11"
				>{formatTick(tick)}</text
			>
		</g>
	{/each}
</g>
