<script lang="ts">
	import type { DomainEvent } from '$lib/types';
	import { accounts } from '$lib/accounts';

	let { event, index, active = false }: { event: DomainEvent; index: number; active: boolean } = $props();

	const typeLabels: Record<string, string> = {
		'investment.made': 'Investment',
		'permit.paid': 'Permit',
		'supply.purchased': 'Supply Purchase',
		'ice.purchased': 'Ice Purchase',
		'sale.completed': 'Sale',
		'spoilage.recorded': 'Spoilage',
		'advertising.purchased': 'Advertising',
	};

	const typeColors: Record<string, string> = {
		'investment.made': '#4ade80',
		'permit.paid': '#a78bfa',
		'supply.purchased': '#f59e0b',
		'ice.purchased': '#38bdf8',
		'sale.completed': '#ffd700',
		'spoilage.recorded': '#ef4444',
		'advertising.purchased': '#fb923c',
	};

	let label = $derived(typeLabels[event.type] || event.type);
	let color = $derived(typeColors[event.type] || '#888');
	let date = $derived(event.data.date as string || '');
	let je = $derived(event.journal_entry);

	function formatAmount(val: unknown): string {
		const n = parseFloat(String(val));
		return isNaN(n) ? '0.00' : n.toFixed(2);
	}

	function describeEvent(e: DomainEvent): string {
		const d = e.data;
		switch (e.type) {
			case 'investment.made':
				return `${d.description} — $${formatAmount(d.amount)}`;
			case 'permit.paid':
				return `${d.description} — $${formatAmount(d.amount)}`;
			case 'supply.purchased': {
				const items = (d.items as Array<{ name: string; cost: string }>) || [];
				const total = items.reduce((s, i) => s + parseFloat(i.cost), 0);
				return `Weekly supply run — $${total.toFixed(2)}`;
			}
			case 'ice.purchased':
				return `Daily ice — $${formatAmount(d.cost)}`;
			case 'sale.completed':
				return `${d.cups} cups @ $${formatAmount(d.price_per_cup)} (${d.weather})`;
			case 'spoilage.recorded':
				return `${d.item} — $${formatAmount(d.amount)} (${d.reason})`;
			case 'advertising.purchased':
				return `${d.description} — $${formatAmount(d.amount)}`;
			default:
				return JSON.stringify(d);
		}
	}

	function accountName(num: string): string {
		return accounts[num]?.name || num;
	}
</script>

<div class="card" class:active id="event-{index}" data-index={index}>
	<div class="header">
		<span class="badge" style="background: {color}20; color: {color}; border-color: {color}40">
			{label}
		</span>
		<span class="date">{date}</span>
		<span class="seq">#{event.sequence}</span>
	</div>

	<p class="description">{describeEvent(event)}</p>

	{#if je}
		<div class="journal">
			<div class="journal-header">Journal Entry</div>
			{#each je.lines as line}
				{@const debit = parseFloat(line.debit)}
				{@const credit = parseFloat(line.credit)}
				{#if debit > 0}
					<div class="journal-line">
						<span class="account-name">{accountName(line.account)}</span>
						<span class="dr">${debit.toFixed(2)}</span>
						<span class="cr"></span>
					</div>
				{:else if credit > 0}
					<div class="journal-line indent">
						<span class="account-name">{accountName(line.account)}</span>
						<span class="dr"></span>
						<span class="cr">${credit.toFixed(2)}</span>
					</div>
				{/if}
			{/each}
		</div>
	{/if}
</div>

<style>
	.card {
		padding: 1.25rem;
		border: 1px solid #222;
		border-radius: 8px;
		background: #111;
		transition: border-color 0.3s, box-shadow 0.3s;
	}

	.card.active {
		border-color: #ffd70060;
		box-shadow: 0 0 20px #ffd70010;
	}

	.header {
		display: flex;
		align-items: center;
		gap: 0.75rem;
		margin-bottom: 0.5rem;
	}

	.badge {
		font-size: 0.7rem;
		font-weight: 600;
		text-transform: uppercase;
		letter-spacing: 0.05em;
		padding: 0.15rem 0.5rem;
		border-radius: 4px;
		border: 1px solid;
	}

	.date {
		font-size: 0.8rem;
		color: #888;
		font-variant-numeric: tabular-nums;
	}

	.seq {
		font-size: 0.7rem;
		color: #555;
		margin-left: auto;
		font-variant-numeric: tabular-nums;
	}

	.description {
		font-size: 0.9rem;
		color: #ccc;
		margin-bottom: 0.75rem;
	}

	.journal {
		background: #0a0a0a;
		border-radius: 6px;
		padding: 0.75rem;
		font-family: 'SF Mono', 'Fira Code', monospace;
		font-size: 0.8rem;
	}

	.journal-header {
		font-size: 0.65rem;
		text-transform: uppercase;
		letter-spacing: 0.08em;
		color: #666;
		margin-bottom: 0.5rem;
	}

	.journal-line {
		display: grid;
		grid-template-columns: 1fr auto auto;
		gap: 1rem;
		padding: 0.2rem 0;
	}

	.journal-line.indent .account-name {
		padding-left: 1rem;
	}

	.account-name { color: #bbb; }
	.dr { color: #4ade80; text-align: right; min-width: 5rem; }
	.cr { color: #ffd700; text-align: right; min-width: 5rem; }
</style>
