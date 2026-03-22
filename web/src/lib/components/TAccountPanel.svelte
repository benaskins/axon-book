<script lang="ts">
	import TAccount from './TAccount.svelte';
	import type { TAccountEntry } from '$lib/types';
	import { accounts, accountGroups } from '$lib/accounts';

	let {
		accountEntries,
		activeEventIndex = -1,
	}: {
		accountEntries: Record<string, TAccountEntry[]>;
		activeEventIndex: number;
	} = $props();

	// Only show accounts that have been touched
	let activeAccounts = $derived(
		Object.keys(accountEntries).filter(num => accountEntries[num].length > 0)
	);

	let groupedAccounts = $derived(
		accountGroups
			.map(g => ({
				...g,
				accounts: activeAccounts
					.filter(num => accounts[num]?.type === g.type)
					.sort()
			}))
			.filter(g => g.accounts.length > 0)
	);
</script>

<div class="panel">
	<h2 class="panel-title">The Ledger</h2>

	{#each groupedAccounts as group}
		<div class="group">
			<h3 class="group-label">{group.label}</h3>
			{#each group.accounts as num}
				<TAccount
					name={accounts[num].name}
					number={num}
					entries={accountEntries[num]}
					highlightIndex={activeEventIndex}
				/>
			{/each}
		</div>
	{/each}

	{#if activeAccounts.length === 0}
		<p class="empty">Scroll through events to build the ledger...</p>
	{/if}
</div>

<style>
	.panel {
		padding: 1rem;
	}

	.panel-title {
		font-size: 1rem;
		font-weight: 600;
		color: #ffd700;
		margin-bottom: 1rem;
		padding-bottom: 0.5rem;
		border-bottom: 1px solid #333;
	}

	.group {
		margin-bottom: 1.25rem;
	}

	.group-label {
		font-size: 0.7rem;
		text-transform: uppercase;
		letter-spacing: 0.1em;
		color: #666;
		margin-bottom: 0.5rem;
	}

	.empty {
		color: #555;
		font-size: 0.85rem;
		font-style: italic;
		text-align: center;
		padding: 2rem 0;
	}
</style>
