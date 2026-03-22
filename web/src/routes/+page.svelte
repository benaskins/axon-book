<script lang="ts">
	import EventCard from '$lib/components/EventCard.svelte';
	import TAccountPanel from '$lib/components/TAccountPanel.svelte';
	import type { DomainEvent, TAccountEntry } from '$lib/types';
	import rawEvents from '$lib/data/events.json';

	const events: DomainEvent[] = rawEvents as DomainEvent[];

	let activeEventIndex = $state(-1);

	// Build cumulative T-account entries up to and including the active event
	let accountEntries = $derived.by(() => {
		const entries: Record<string, TAccountEntry[]> = {};
		const limit = activeEventIndex + 1;

		for (let i = 0; i < limit && i < events.length; i++) {
			const ev = events[i];
			const je = ev.journal_entry;
			if (!je) continue;

			for (const line of je.lines) {
				const debit = parseFloat(line.debit);
				const credit = parseFloat(line.credit);

				if (!entries[line.account]) entries[line.account] = [];

				if (debit > 0) {
					entries[line.account].push({
						eventIndex: i,
						amount: debit,
						side: 'debit',
						description: je.description,
					});
				}
				if (credit > 0) {
					entries[line.account].push({
						eventIndex: i,
						amount: credit,
						side: 'credit',
						description: je.description,
					});
				}
			}
		}
		return entries;
	});

	// Intersection observer to track which event card is in view
	$effect(() => {
		const observers: IntersectionObserver[] = [];
		for (let i = 0; i < events.length; i++) {
			const el = document.getElementById(`event-${i}`);
			if (!el) continue;
			const observer = new IntersectionObserver(
				(entries) => {
					for (const entry of entries) {
						if (entry.isIntersecting) {
							activeEventIndex = i;
						}
					}
				},
				{ threshold: 0.5, rootMargin: '-20% 0px -20% 0px' }
			);
			observer.observe(el);
			observers.push(observer);
		}
		return () => observers.forEach((o) => o.disconnect());
	});
</script>

<section class="hero">
	<h1 class="title">The Lemonade Stand</h1>
	<p class="subtitle">An event-sourced story told through double-entry bookkeeping</p>
	<p class="intro">
		Every business starts with events. A sale, a purchase, a loss.
		Each event triggers a journal entry. Each entry touches two sides of the ledger.
		Scroll to watch it happen.
	</p>
	<div class="scroll-hint">&#8595;</div>
</section>

<div class="layout">
	<div class="events-column">
		{#each events as event, i}
			<EventCard {event} index={i} active={i === activeEventIndex} />
		{/each}
		<div class="end-marker">
			<p>30 days. {events.length} events. Every dollar accounted for.</p>
		</div>
	</div>

	<div class="ledger-column">
		<div class="ledger-sticky">
			<TAccountPanel {accountEntries} activeEventIndex={activeEventIndex} />
		</div>
	</div>
</div>

<style>
	.hero {
		display: flex;
		flex-direction: column;
		align-items: center;
		justify-content: center;
		min-height: 80vh;
		padding: 2rem;
		text-align: center;
	}

	.title {
		font-size: clamp(2.5rem, 6vw, 4rem);
		font-weight: 700;
		color: #ffd700;
		letter-spacing: -0.02em;
		margin-bottom: 0.75rem;
	}

	.subtitle {
		font-size: clamp(0.9rem, 2vw, 1.2rem);
		color: #888;
		font-weight: 300;
		margin-bottom: 2rem;
	}

	.intro {
		max-width: 50ch;
		color: #999;
		font-size: 0.95rem;
		line-height: 1.7;
	}

	.scroll-hint {
		margin-top: 3rem;
		font-size: 1.5rem;
		color: #444;
		animation: bounce 2s infinite;
	}

	@keyframes bounce {
		0%, 100% { transform: translateY(0); }
		50% { transform: translateY(8px); }
	}

	.layout {
		display: grid;
		grid-template-columns: 1fr 340px;
		gap: 0;
		max-width: 1200px;
		margin: 0 auto;
		padding: 0 1rem;
	}

	.events-column {
		display: flex;
		flex-direction: column;
		gap: 1rem;
		padding: 1rem 1.5rem 40vh 0;
	}

	.ledger-column {
		border-left: 1px solid #222;
	}

	.ledger-sticky {
		position: sticky;
		top: 0;
		max-height: 100vh;
		overflow-y: auto;
		scrollbar-width: thin;
		scrollbar-color: #333 transparent;
	}

	.end-marker {
		text-align: center;
		padding: 4rem 0;
		color: #555;
		font-size: 0.9rem;
	}

	@media (max-width: 900px) {
		.layout {
			grid-template-columns: 1fr;
		}

		.ledger-column {
			display: none;
		}
	}
</style>
