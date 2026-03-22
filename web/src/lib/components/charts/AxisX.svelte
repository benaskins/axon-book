<script lang="ts">
	import { getContext } from 'svelte';

	let { tickCount = 5, formatTick = (d: any) => String(d) } = $props();

	const { xScale, yRange } = getContext('LayerCake') as any;

	let ticks = $derived.by(() => {
		const scale = $xScale;
		if (!scale || !scale.ticks) return [];
		return scale.ticks(tickCount);
	});

	let yBottom = $derived.by(() => {
		const range = $yRange;
		return range ? range[0] : 0;
	});
</script>

<g class="axis-x">
	{#each ticks as tick}
		<g transform="translate({$xScale(tick)}, {yBottom})">
			<line y1="0" y2="6" stroke="#444" />
			<text y="20" text-anchor="middle" fill="#888" font-size="11">{formatTick(tick)}</text>
		</g>
	{/each}
</g>
