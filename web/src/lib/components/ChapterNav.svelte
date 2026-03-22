<script lang="ts">
	let { chapters, activeChapter }: { chapters: string[]; activeChapter: string } = $props();

	function scrollTo(id: string) {
		const el = document.getElementById(id);
		if (el) el.scrollIntoView({ behavior: 'smooth' });
	}
</script>

<nav class="chapter-nav" aria-label="Chapter navigation">
	{#each chapters as chapter}
		<button
			class="dot"
			class:active={activeChapter === chapter}
			onclick={() => scrollTo(chapter)}
			aria-label="Go to {chapter}"
		>
			<span class="dot-inner"></span>
		</button>
	{/each}
</nav>

<style>
	.chapter-nav {
		position: fixed;
		right: 1.5rem;
		top: 50%;
		transform: translateY(-50%);
		display: flex;
		flex-direction: column;
		gap: 0.75rem;
		z-index: 100;
	}

	.dot {
		background: none;
		border: none;
		cursor: pointer;
		padding: 4px;
		display: flex;
		align-items: center;
		justify-content: center;
	}

	.dot-inner {
		width: 10px;
		height: 10px;
		border-radius: 50%;
		background: #333;
		transition: background 0.3s ease, transform 0.3s ease;
	}

	.dot.active .dot-inner {
		background: #ffd700;
		transform: scale(1.4);
	}

	.dot:hover .dot-inner {
		background: #ffd700;
		opacity: 0.7;
	}

	@media (max-width: 768px) {
		.chapter-nav {
			right: 0.75rem;
		}

		.dot-inner {
			width: 8px;
			height: 8px;
		}
	}
</style>
