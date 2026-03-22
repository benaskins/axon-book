<script lang="ts">
	import type { Snippet } from 'svelte';

	let { id, title, children }: { id: string; title: string; children: Snippet } = $props();

	let element: HTMLElement | undefined = $state();
	let visible = $state(false);

	$effect(() => {
		if (!element) return;
		const observer = new IntersectionObserver(
			(entries) => {
				for (const entry of entries) {
					if (entry.isIntersecting) {
						visible = true;
					}
				}
			},
			{ threshold: 0.3 }
		);
		observer.observe(element);
		return () => observer.disconnect();
	});
</script>

<section {id} class="chapter" class:visible bind:this={element}>
	<div class="chapter-inner">
		<div class="narrative">
			<h2 class="chapter-title">{title}</h2>
			{@render children()}
		</div>
	</div>
</section>

<style>
	.chapter {
		min-height: 100vh;
		display: flex;
		align-items: center;
		padding: 4rem 2rem;
		opacity: 0;
		transform: translateY(40px);
		transition: opacity 0.8s ease, transform 0.8s ease;
	}

	.chapter.visible {
		opacity: 1;
		transform: translateY(0);
	}

	.chapter-inner {
		max-width: 1200px;
		margin: 0 auto;
		width: 100%;
		display: grid;
		grid-template-columns: 1fr 1fr;
		gap: 4rem;
		align-items: center;
	}

	.narrative {
		grid-column: 1;
	}

	.chapter-title {
		font-size: 2.5rem;
		font-weight: 700;
		color: #ffd700;
		margin-bottom: 1.5rem;
		line-height: 1.2;
	}

	:global(.chapter .narrative p) {
		font-size: 1.1rem;
		font-weight: 300;
		line-height: 1.8;
		color: #e8e8e8;
		margin-bottom: 1rem;
	}

	:global(.chapter .chart-panel) {
		grid-column: 2;
		grid-row: 1;
		background: #1a1a1a;
		border-radius: 12px;
		padding: 2rem;
		min-height: 350px;
	}

	@media (max-width: 768px) {
		.chapter-inner {
			grid-template-columns: 1fr;
			gap: 2rem;
		}

		:global(.chapter .chart-panel) {
			grid-column: 1;
			grid-row: auto;
		}

		.chapter-title {
			font-size: 1.8rem;
		}

		.chapter {
			padding: 3rem 1.5rem;
		}
	}
</style>
