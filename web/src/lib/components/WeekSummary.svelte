<script lang="ts">
	import type { DomainEvent } from '$lib/types';

	let { events, weekNumber, weekStart, weekEnd }: {
		events: DomainEvent[];
		weekNumber: number;
		weekStart: string;
		weekEnd: string;
	} = $props();

	// Compute weekly stats from domain events
	let stats = $derived.by(() => {
		let cups = 0;
		let revenue = 0;
		let supplyCost = 0;
		let iceCost = 0;
		let spoilage = 0;
		let advertising = 0;
		let permit = 0;
		let salesDays = 0;
		let weatherCounts: Record<string, number> = {};
		let bestDay = { cups: 0, date: '' };

		for (const e of events) {
			const d = e.data;
			switch (e.type) {
				case 'sale.completed': {
					const c = d.cups as number;
					const price = parseFloat(d.price_per_cup as string);
					cups += c;
					revenue += c * price;
					salesDays++;
					const w = d.weather as string;
					weatherCounts[w] = (weatherCounts[w] || 0) + 1;
					if (c > bestDay.cups) {
						bestDay = { cups: c, date: d.date as string };
					}
					break;
				}
				case 'supply.purchased': {
					const items = d.items as Array<{ cost: string }>;
					supplyCost += items.reduce((s, i) => s + parseFloat(i.cost), 0);
					break;
				}
				case 'ice.purchased':
					iceCost += parseFloat(d.cost as string);
					break;
				case 'spoilage.recorded':
					spoilage += parseFloat(d.amount as string);
					break;
				case 'advertising.purchased':
					advertising += parseFloat(d.amount as string);
					break;
				case 'permit.paid':
					permit += parseFloat(d.amount as string);
					break;
			}
		}

		const totalCost = supplyCost + iceCost + spoilage + advertising + permit;
		const dominantWeather = Object.entries(weatherCounts)
			.sort((a, b) => b[1] - a[1])[0]?.[0] || '';

		return { cups, revenue, supplyCost, iceCost, spoilage, advertising, permit, totalCost, salesDays, dominantWeather, bestDay };
	});

	let insight = $derived.by(() => {
		const s = stats;
		const lines: string[] = [];

		if (weekNumber === 0) {
			// First week — the setup
			if (s.cups === 0) {
				lines.push("The stand isn't open yet. Just getting set up — investment in, permit paid, supplies stocked.");
			} else {
				lines.push(`Opening week. ${s.cups} cups sold across ${s.salesDays} day${s.salesDays !== 1 ? 's' : ''}.`);
			}
			if (s.supplyCost > 0) {
				lines.push(`Stocked up on $${s.supplyCost.toFixed(0)} worth of supplies.`);
			}
			if (s.permit > 0) {
				lines.push(`Stand permit: $${s.permit.toFixed(0)}.`);
			}
		} else if (s.salesDays === 0) {
			lines.push("No sales this week — weather or timing kept the stand closed.");
			if (s.spoilage > 0) {
				lines.push(`Lost $${s.spoilage.toFixed(2)} to spoilage while shut.`);
			}
		} else {
			lines.push(`${s.cups} cups across ${s.salesDays} day${s.salesDays !== 1 ? 's' : ''} — $${s.revenue.toFixed(2)} in revenue.`);

			if (s.bestDay.cups > 0 && s.salesDays > 1) {
				lines.push(`Best day: ${s.bestDay.cups} cups on ${formatShortDate(s.bestDay.date)}.`);
			}

			if (s.dominantWeather) {
				const weatherNote: Record<string, string> = {
					'hot': 'Hot days drove the numbers up.',
					'mild': 'Mild weather — steady but not spectacular.',
					'cold': 'Cold days kept foot traffic low.',
				};
				if (weatherNote[s.dominantWeather]) {
					lines.push(weatherNote[s.dominantWeather]);
				}
			}

			if (s.spoilage > 0) {
				lines.push(`$${s.spoilage.toFixed(2)} lost to spoilage.`);
			}

			const margin = s.revenue - s.totalCost;
			if (s.revenue > 0) {
				if (margin > 0) {
					lines.push(`Net for the week: +$${margin.toFixed(2)}.`);
				} else {
					lines.push(`Net for the week: -$${Math.abs(margin).toFixed(2)}. Costs outpaced sales.`);
				}
			}
		}

		return lines;
	});

	function formatShortDate(d: string): string {
		const date = new Date(d + 'T00:00:00');
		return date.toLocaleDateString('en', { weekday: 'short', month: 'short', day: 'numeric' });
	}

	function formatRange(start: string, end: string): string {
		const s = new Date(start + 'T00:00:00');
		const e = new Date(end + 'T00:00:00');
		const sMonth = s.toLocaleDateString('en', { month: 'short' });
		const eMonth = e.toLocaleDateString('en', { month: 'short' });
		if (sMonth === eMonth) {
			return `${sMonth} ${s.getDate()}–${e.getDate()}`;
		}
		return `${sMonth} ${s.getDate()} – ${eMonth} ${e.getDate()}`;
	}
</script>

<div class="summary">
	<div class="summary-header">
		<span class="week-label">Week {weekNumber + 1}</span>
		<span class="date-range">{formatRange(weekStart, weekEnd)}</span>
	</div>

	{#if stats.salesDays > 0}
		<div class="stats-row">
			<div class="stat">
				<span class="stat-value">{stats.cups}</span>
				<span class="stat-label">cups</span>
			</div>
			<div class="stat">
				<span class="stat-value">${stats.revenue.toFixed(0)}</span>
				<span class="stat-label">revenue</span>
			</div>
			<div class="stat">
				<span class="stat-value">${stats.totalCost.toFixed(0)}</span>
				<span class="stat-label">costs</span>
			</div>
		</div>
	{/if}

	<div class="insight">
		{#each insight as line}
			<p>{line}</p>
		{/each}
	</div>
</div>

<style>
	.summary {
		padding: 1.25rem 1.5rem;
		border-left: 3px solid #ffd70040;
		background: linear-gradient(135deg, #ffd70008, transparent);
		border-radius: 0 8px 8px 0;
		margin: 0.5rem 0;
	}

	.summary-header {
		display: flex;
		align-items: baseline;
		gap: 0.75rem;
		margin-bottom: 0.75rem;
	}

	.week-label {
		font-size: 0.75rem;
		font-weight: 700;
		text-transform: uppercase;
		letter-spacing: 0.08em;
		color: #ffd700;
	}

	.date-range {
		font-size: 0.8rem;
		color: #666;
	}

	.stats-row {
		display: flex;
		gap: 1.5rem;
		margin-bottom: 0.75rem;
	}

	.stat {
		display: flex;
		flex-direction: column;
	}

	.stat-value {
		font-size: 1.1rem;
		font-weight: 600;
		color: #e8e8e8;
		font-variant-numeric: tabular-nums;
	}

	.stat-label {
		font-size: 0.65rem;
		text-transform: uppercase;
		letter-spacing: 0.06em;
		color: #666;
	}

	.insight p {
		font-size: 0.85rem;
		color: #aaa;
		line-height: 1.6;
		margin-bottom: 0.25rem;
	}

	.insight p:last-child {
		margin-bottom: 0;
	}
</style>
