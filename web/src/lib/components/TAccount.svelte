<script lang="ts">
	import type { TAccountEntry } from '$lib/types';

	let {
		name,
		number,
		entries,
		highlightIndex = -1,
	}: {
		name: string;
		number: string;
		entries: TAccountEntry[];
		highlightIndex: number;
	} = $props();

	let totalDebit = $derived(entries.filter(e => e.side === 'debit').reduce((s, e) => s + e.amount, 0));
	let totalCredit = $derived(entries.filter(e => e.side === 'credit').reduce((s, e) => s + e.amount, 0));
	let balance = $derived(Math.abs(totalDebit - totalCredit));
	let balanceSide = $derived(totalDebit >= totalCredit ? 'debit' : 'credit');
</script>

<div class="taccount">
	<div class="taccount-header">
		<span class="taccount-number">{number}</span>
		<span class="taccount-name">{name}</span>
	</div>
	<div class="t-body">
		<div class="t-side debit-side">
			<div class="t-side-header">DR</div>
			{#each entries.filter(e => e.side === 'debit') as entry}
				<div class="t-entry" class:highlight={entry.eventIndex === highlightIndex}>
					${entry.amount.toFixed(2)}
				</div>
			{/each}
			<div class="t-total" class:balance-side={balanceSide === 'debit'}>
				{#if balanceSide === 'debit'}
					<strong>${balance.toFixed(2)}</strong>
				{/if}
			</div>
		</div>
		<div class="t-divider"></div>
		<div class="t-side credit-side">
			<div class="t-side-header">CR</div>
			{#each entries.filter(e => e.side === 'credit') as entry}
				<div class="t-entry" class:highlight={entry.eventIndex === highlightIndex}>
					${entry.amount.toFixed(2)}
				</div>
			{/each}
			<div class="t-total" class:balance-side={balanceSide === 'credit'}>
				{#if balanceSide === 'credit'}
					<strong>${balance.toFixed(2)}</strong>
				{/if}
			</div>
		</div>
	</div>
</div>

<style>
	.taccount {
		margin-bottom: 0.75rem;
	}

	.taccount-header {
		display: flex;
		gap: 0.5rem;
		align-items: baseline;
		padding: 0.25rem 0;
		border-bottom: 2px solid #444;
		margin-bottom: 0.25rem;
	}

	.taccount-number {
		font-size: 0.65rem;
		color: #666;
		font-variant-numeric: tabular-nums;
	}

	.taccount-name {
		font-size: 0.8rem;
		font-weight: 600;
		color: #ddd;
	}

	.t-body {
		display: grid;
		grid-template-columns: 1fr 1px 1fr;
		font-size: 0.75rem;
		font-variant-numeric: tabular-nums;
		font-family: 'SF Mono', 'Fira Code', monospace;
	}

	.t-side {
		padding: 0.15rem 0.4rem;
	}

	.t-side-header {
		font-size: 0.6rem;
		color: #666;
		text-align: center;
		margin-bottom: 0.15rem;
	}

	.debit-side { text-align: right; }
	.credit-side { text-align: right; }

	.t-divider {
		background: #333;
	}

	.t-entry {
		padding: 0.1rem 0;
		color: #bbb;
		transition: color 0.3s, background 0.3s;
	}

	.t-entry.highlight {
		color: #ffd700;
		background: #ffd70010;
	}

	.t-total {
		border-top: 1px solid #333;
		padding-top: 0.15rem;
		margin-top: 0.15rem;
		min-height: 1.2em;
		color: #888;
	}

	.t-total.balance-side {
		color: #4ade80;
	}
</style>
