<script lang="ts">
	let { children } = $props();
</script>

<svelte:head>
	<title>The Lemonade Stand</title>
	<meta name="description" content="An event-sourced story told through double-entry bookkeeping" />
</svelte:head>

{@render children()}

<style>
	:global(*) {
		margin: 0;
		padding: 0;
		box-sizing: border-box;
	}

	:global(body) {
		background-color: #0a0a0a;
		color: #e8e8e8;
		font-family: 'Inter', 'Helvetica Neue', Arial, sans-serif;
		line-height: 1.6;
		-webkit-font-smoothing: antialiased;
		-moz-osx-font-smoothing: grayscale;
	}

	:global(a) {
		color: #ffd700;
		text-decoration: none;
	}

	:global(a:hover) {
		text-decoration: underline;
	}
</style>
