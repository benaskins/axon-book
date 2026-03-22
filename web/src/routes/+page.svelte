<script lang="ts">
	import { LayerCake, Svg } from 'layercake';
	import { scaleTime, scaleLinear } from 'd3-scale';
	import Chapter from '$lib/components/Chapter.svelte';
	import ChapterNav from '$lib/components/ChapterNav.svelte';
	import Line from '$lib/components/charts/Line.svelte';
	import Area from '$lib/components/charts/Area.svelte';
	import ColoredLine from '$lib/components/charts/ColoredLine.svelte';
	import StackedArea from '$lib/components/charts/StackedArea.svelte';
	import AxisX from '$lib/components/charts/AxisX.svelte';
	import AxisY from '$lib/components/charts/AxisY.svelte';
	import Scatter from '$lib/components/charts/Scatter.svelte';
	import {
		fetchDailySummaries,
		fetchTrialBalance,
		fetchProfitAndLoss,
		type DailySummary,
		type TrialBalance,
		type ProfitAndLoss
	} from '$lib/api';

	const chapterIds = [
		'opening-day',
		'spring-awakening',
		'summer-rush',
		'cost-of-lemonade',
		'spoilage',
		'winding-down',
		'the-books'
	];

	let loading = $state(true);
	let error = $state('');
	let dailyData: DailySummary[] = $state([]);
	let trialBalance: TrialBalance | null = $state(null);
	let profitAndLoss: ProfitAndLoss | null = $state(null);
	let activeChapter = $state('opening-day');

	// Fetch all data on mount
	$effect(() => {
		Promise.all([
			fetchDailySummaries('2025-01-01', '2025-12-31'),
			fetchTrialBalance(),
			fetchProfitAndLoss('2025-01-01', '2025-12-31')
		])
			.then(([daily, tb, pl]) => {
				dailyData = daily;
				trialBalance = tb;
				profitAndLoss = pl;
				loading = false;
			})
			.catch((e) => {
				error = e.message;
				loading = false;
			});
	});

	// Set up chapter intersection observers for nav tracking
	$effect(() => {
		if (loading) return;
		const observers: IntersectionObserver[] = [];
		for (const id of chapterIds) {
			const el = document.getElementById(id);
			if (!el) continue;
			const observer = new IntersectionObserver(
				(entries) => {
					for (const entry of entries) {
						if (entry.isIntersecting) {
							activeChapter = id;
						}
					}
				},
				{ threshold: 0.3 }
			);
			observer.observe(el);
			observers.push(observer);
		}
		return () => observers.forEach((o) => o.disconnect());
	});

	// Derived data for charts
	let chartData = $derived(
		dailyData.map((d) => ({
			...d,
			dateObj: new Date(d.date),
			revenueNum: parseFloat(d.revenue),
			cupsNum: d.cups_sold,
			cogsLemons: parseFloat(d.cogs_lemons),
			cogsSugar: parseFloat(d.cogs_sugar),
			cogsCups: parseFloat(d.cogs_cups),
			cogsIce: parseFloat(d.cogs_ice),
			spoilageNum: parseFloat(d.spoilage),
			iceCostNum: parseFloat(d.ice_cost)
		}))
	);

	let springData = $derived(
		chartData.filter((d) => {
			const m = d.dateObj.getMonth();
			return m >= 2 && m <= 4; // Mar-May
		})
	);

	let summerHighlightData = $derived(chartData.filter((d) => d.dateObj.getMonth() <= 8)); // Jan-Sep

	let spoilageData = $derived(chartData.filter((d) => d.spoilageNum > 0));

	let windingData = $derived(
		chartData.filter((d) => {
			const m = d.dateObj.getMonth();
			return m >= 7 && m <= 10; // Aug-Nov
		})
	);

	// Stacked area series config
	const cogsSeries = [
		{ key: 'cogsLemons', color: '#ffd700' },
		{ key: 'cogsSugar', color: '#ff8c00' },
		{ key: 'cogsCups', color: '#a78bfa' },
		{ key: 'cogsIce', color: '#4a9eff' }
	];

	// Max COGS stacked value
	let maxCogs = $derived(
		Math.max(
			...chartData.map(
				(d) => d.cogsLemons + d.cogsSugar + d.cogsCups + d.cogsIce
			),
			1
		)
	);

	function weatherColor(d: { weather: string }): string {
		switch (d.weather) {
			case 'hot':
				return '#ff8c00';
			case 'warm':
				return '#ffd700';
			case 'mild':
				return '#4ade80';
			case 'cool':
				return '#4a9eff';
			case 'cold':
				return '#6366f1';
			case 'rainy':
				return '#64748b';
			default:
				return '#a0a0a0';
		}
	}

	function formatMonth(d: Date): string {
		return d.toLocaleString('en', { month: 'short' });
	}

	function formatDollar(d: number): string {
		return '$' + d.toFixed(0);
	}

	// Summary stats
	let totalRevenue = $derived(chartData.reduce((sum, d) => sum + d.revenueNum, 0));
	let totalCups = $derived(chartData.reduce((sum, d) => sum + d.cupsNum, 0));
	let operatingDays = $derived(chartData.filter((d) => d.cupsNum > 0).length);
	let peakDay = $derived(
		chartData.reduce(
			(max, d) => (d.cupsNum > max.cupsNum ? d : max),
			chartData[0] || { cupsNum: 0, revenueNum: 0, date: '' }
		)
	);
</script>

{#if loading}
	<section class="hero">
		<h1 class="title">The Lemonade Stand</h1>
		<p class="subtitle">Loading the books...</p>
		<div class="loader"></div>
	</section>
{:else if error}
	<section class="hero">
		<h1 class="title">The Lemonade Stand</h1>
		<p class="subtitle error">{error}</p>
	</section>
{:else}
	<ChapterNav chapters={chapterIds} {activeChapter} />

	<!-- Hero -->
	<section class="hero">
		<h1 class="title">The Lemonade Stand</h1>
		<p class="subtitle">An event-sourced story told through double-entry bookkeeping</p>
		<div class="scroll-hint">
			<span class="arrow">&#8595;</span>
		</div>
	</section>

	<!-- Chapter 1: Opening Day -->
	<Chapter id="opening-day" title="Opening Day">
		<p>
			Every business starts with a single entry. On January 1st, 2025, our lemonade stand owner
			invested $5,000 of personal savings.
		</p>
		<p>
			In double-entry bookkeeping, this means two things happened at once: cash increased, and so
			did the owner's stake in the business. The books balance from the very first moment.
		</p>
		<div class="chart-panel">
			<div class="opening-display">
				<div class="stat-block">
					<span class="stat-label">Initial Investment</span>
					<span class="stat-value gold">$5,000</span>
				</div>
				<div class="ledger-entry">
					<div class="entry-row">
						<span class="account">Cash</span>
						<span class="debit">$5,000</span>
						<span class="credit">&mdash;</span>
					</div>
					<div class="entry-row indent">
						<span class="account">Owner's Equity</span>
						<span class="debit">&mdash;</span>
						<span class="credit">$5,000</span>
					</div>
				</div>
				<div class="balance-note">Assets = Liabilities + Equity</div>
			</div>
		</div>
	</Chapter>

	<!-- Chapter 2: Spring Awakening -->
	<Chapter id="spring-awakening" title="Spring Awakening">
		<p>
			March brings the first customers. Sales are modest — a few cups on warm afternoons, nothing
			on cold days. But the stand is open, the lemons are fresh, and the ledger is starting to fill.
		</p>
		<div class="chart-panel">
			{#if springData.length > 0}
				<div class="chart-container">
					<LayerCake
						data={springData}
						x={(d) => d.dateObj}
						y={(d) => d.revenueNum}
						xScale={scaleTime()}
						yScale={scaleLinear()}
						padding={{ top: 20, right: 15, bottom: 30, left: 50 }}
					>
						<Svg>
							<AxisX formatTick={formatMonth} tickCount={3} />
							<AxisY formatTick={formatDollar} tickCount={4} />
							<Area fill="#ffd700" opacity={0.15} />
							<Line stroke="#ffd700" strokeWidth={2} />
						</Svg>
					</LayerCake>
				</div>
				<div class="chart-caption">Daily revenue, March&ndash;May 2025</div>
			{/if}
		</div>
	</Chapter>

	<!-- Chapter 3: Summer Rush -->
	<Chapter id="summer-rush" title="Summer Rush">
		<p>
			Then summer hits. Hot days mean long lines. At peak, we're selling over {peakDay?.cupsNum ?? 60} cups
			a day at $3.50 each. Revenue spikes with the temperature — every scorching afternoon is a
			record-breaking day.
		</p>
		<div class="chart-panel">
			{#if summerHighlightData.length > 0}
				<div class="chart-container">
					<LayerCake
						data={summerHighlightData}
						x={(d) => d.dateObj}
						y={(d) => d.revenueNum}
						xScale={scaleTime()}
						yScale={scaleLinear()}
						padding={{ top: 20, right: 15, bottom: 30, left: 50 }}
					>
						<Svg>
							<AxisX formatTick={formatMonth} tickCount={6} />
							<AxisY formatTick={formatDollar} tickCount={5} />
							<ColoredLine colorFn={weatherColor} strokeWidth={2} />
						</Svg>
					</LayerCake>
				</div>
				<div class="weather-legend">
					<span class="legend-item"><span class="swatch" style="background:#ff8c00"></span> Hot</span>
					<span class="legend-item"><span class="swatch" style="background:#ffd700"></span> Warm</span>
					<span class="legend-item"><span class="swatch" style="background:#4ade80"></span> Mild</span>
					<span class="legend-item"><span class="swatch" style="background:#4a9eff"></span> Cool</span>
					<span class="legend-item"><span class="swatch" style="background:#64748b"></span> Rainy</span>
				</div>
			{/if}
		</div>
	</Chapter>

	<!-- Chapter 4: The Cost of Lemonade -->
	<Chapter id="cost-of-lemonade" title="The Cost of Lemonade">
		<p>
			Revenue tells one story. Costs tell another. Every cup requires lemons, sugar, a cup, and
			ice. On hot days, ice costs surge — we burn through twice as much.
		</p>
		<p>The stacked costs reveal the real economics of a cup of lemonade.</p>
		<div class="chart-panel">
			{#if chartData.length > 0}
				<div class="chart-container">
					<LayerCake
						data={chartData}
						x={(d) => d.dateObj}
						y={(d) => d.cogsLemons + d.cogsSugar + d.cogsCups + d.cogsIce}
						xScale={scaleTime()}
						yScale={scaleLinear()}
						yDomain={[0, maxCogs]}
						padding={{ top: 20, right: 15, bottom: 30, left: 50 }}
					>
						<Svg>
							<AxisX formatTick={formatMonth} tickCount={6} />
							<AxisY formatTick={formatDollar} tickCount={5} />
							<StackedArea series={cogsSeries} />
						</Svg>
					</LayerCake>
				</div>
				<div class="weather-legend">
					<span class="legend-item"><span class="swatch" style="background:#ffd700"></span> Lemons</span>
					<span class="legend-item"><span class="swatch" style="background:#ff8c00"></span> Sugar</span>
					<span class="legend-item"><span class="swatch" style="background:#a78bfa"></span> Cups</span>
					<span class="legend-item"><span class="swatch" style="background:#4a9eff"></span> Ice</span>
				</div>
			{/if}
		</div>
	</Chapter>

	<!-- Chapter 5: When Life Gives You Spoilage -->
	<Chapter id="spoilage" title="When Life Gives You Spoilage">
		<p>
			Not every day goes to plan. Rainy days close the stand and melt the ice. Lemons left too
			long go overripe. These small losses accumulate — the cost of doing business with perishable
			goods.
		</p>
		<div class="chart-panel">
			{#if spoilageData.length > 0}
				<div class="chart-container">
					<LayerCake
						data={spoilageData}
						x={(d) => d.dateObj}
						y={(d) => d.spoilageNum}
						xScale={scaleTime()}
						yScale={scaleLinear()}
						padding={{ top: 20, right: 15, bottom: 30, left: 50 }}
					>
						<Svg>
							<AxisX formatTick={formatMonth} tickCount={6} />
							<AxisY formatTick={formatDollar} tickCount={4} />
							<Scatter fill="#ef4444" radius={4} opacity={0.8} />
						</Svg>
					</LayerCake>
				</div>
				<div class="spoilage-total">
					Total spoilage: <strong>${spoilageData.reduce((s, d) => s + d.spoilageNum, 0).toFixed(2)}</strong>
				</div>
			{/if}
		</div>
	</Chapter>

	<!-- Chapter 6: Winding Down -->
	<Chapter id="winding-down" title="Winding Down">
		<p>
			September's warmth fades into October's chill. Sales taper, mirroring spring's slow start.
			By the end of October, the stand closes for the season.
		</p>
		<div class="chart-panel">
			{#if windingData.length > 0}
				<div class="chart-container">
					<LayerCake
						data={windingData}
						x={(d) => d.dateObj}
						y={(d) => d.revenueNum}
						xScale={scaleTime()}
						yScale={scaleLinear()}
						padding={{ top: 20, right: 15, bottom: 30, left: 50 }}
					>
						<Svg>
							<AxisX formatTick={formatMonth} tickCount={4} />
							<AxisY formatTick={formatDollar} tickCount={5} />
							<Area fill="#ffd700" opacity={0.1} />
							<Line stroke="#ffd700" strokeWidth={2} />
						</Svg>
					</LayerCake>
				</div>
				<div class="chart-caption">The autumn taper — August through October</div>
			{/if}
		</div>
	</Chapter>

	<!-- Chapter 7: The Books -->
	<Chapter id="the-books" title="The Books">
		<p>
			And now, the moment of truth. After {operatingDays} operating days, hundreds of journal entries,
			and {totalCups.toLocaleString()} cups of lemonade — do the books balance?
		</p>
		<p>Every debit has a credit. Every dollar is accounted for.</p>
		<div class="chart-panel books-panel">
			{#if profitAndLoss}
				<div class="financial-summary">
					<h3 class="section-title">Profit &amp; Loss</h3>
					<div class="pl-section">
						<h4 class="pl-heading revenue-heading">Revenue</h4>
						{#each profitAndLoss.revenue as item}
							<div class="pl-row">
								<span class="pl-account">{item.account}</span>
								<span class="pl-amount">${parseFloat(item.net).toLocaleString(undefined, { minimumFractionDigits: 2, maximumFractionDigits: 2 })}</span>
							</div>
						{/each}
					</div>
					<div class="pl-section">
						<h4 class="pl-heading expense-heading">Expenses</h4>
						{#each profitAndLoss.expenses as item}
							<div class="pl-row">
								<span class="pl-account">{item.account}</span>
								<span class="pl-amount">${parseFloat(item.net).toLocaleString(undefined, { minimumFractionDigits: 2, maximumFractionDigits: 2 })}</span>
							</div>
						{/each}
					</div>
					<div class="pl-total">
						<span class="pl-account">Net Income</span>
						<span class="pl-amount net-income">${parseFloat(profitAndLoss.net_income).toLocaleString(undefined, { minimumFractionDigits: 2, maximumFractionDigits: 2 })}</span>
					</div>
				</div>
			{/if}
			{#if trialBalance}
				<div class="trial-balance">
					<h3 class="section-title">Trial Balance</h3>
					<div class="tb-header">
						<span>Account</span>
						<span>Debits</span>
						<span>Credits</span>
					</div>
					{#each trialBalance.balances as row}
						<div class="tb-row">
							<span class="tb-account">{row.account}</span>
							<span class="tb-amount">${parseFloat(row.debits).toLocaleString(undefined, { minimumFractionDigits: 2, maximumFractionDigits: 2 })}</span>
							<span class="tb-amount">${parseFloat(row.credits).toLocaleString(undefined, { minimumFractionDigits: 2, maximumFractionDigits: 2 })}</span>
						</div>
					{/each}
					<div class="tb-total">
						<span>Totals</span>
						<span class="tb-amount">${parseFloat(trialBalance.total_debits).toLocaleString(undefined, { minimumFractionDigits: 2, maximumFractionDigits: 2 })}</span>
						<span class="tb-amount">${parseFloat(trialBalance.total_credits).toLocaleString(undefined, { minimumFractionDigits: 2, maximumFractionDigits: 2 })}</span>
					</div>
					<div class="balance-status" class:balanced={trialBalance.in_balance}>
						{trialBalance.in_balance ? 'The books balance.' : 'Out of balance!'}
					</div>
				</div>
			{/if}
		</div>
	</Chapter>
{/if}

<style>
	/* Hero */
	.hero {
		display: flex;
		flex-direction: column;
		align-items: center;
		justify-content: center;
		min-height: 100vh;
		padding: 2rem;
		text-align: center;
	}

	.title {
		font-size: clamp(2.5rem, 6vw, 5rem);
		font-weight: 700;
		color: #ffd700;
		letter-spacing: -0.02em;
		margin-bottom: 1rem;
	}

	.subtitle {
		font-size: clamp(1rem, 2vw, 1.5rem);
		color: #a0a0a0;
		font-weight: 300;
		max-width: 40ch;
	}

	.subtitle.error {
		color: #ef4444;
	}

	.scroll-hint {
		margin-top: 3rem;
		animation: bounce 2s infinite;
	}

	.arrow {
		font-size: 2rem;
		color: #555;
	}

	@keyframes bounce {
		0%, 100% { transform: translateY(0); }
		50% { transform: translateY(10px); }
	}

	/* Loader */
	.loader {
		margin-top: 2rem;
		width: 40px;
		height: 40px;
		border: 3px solid #333;
		border-top-color: #ffd700;
		border-radius: 50%;
		animation: spin 0.8s linear infinite;
	}

	@keyframes spin {
		to { transform: rotate(360deg); }
	}

	/* Chart panels */
	.chart-container {
		width: 100%;
		height: 280px;
	}

	.chart-caption {
		text-align: center;
		color: #888;
		font-size: 0.85rem;
		margin-top: 0.75rem;
	}

	/* Weather legend */
	.weather-legend {
		display: flex;
		gap: 1rem;
		justify-content: center;
		margin-top: 0.75rem;
		flex-wrap: wrap;
	}

	.legend-item {
		display: flex;
		align-items: center;
		gap: 0.35rem;
		font-size: 0.8rem;
		color: #888;
	}

	.swatch {
		display: inline-block;
		width: 10px;
		height: 10px;
		border-radius: 50%;
	}

	/* Chapter 1: Opening Day */
	.opening-display {
		display: flex;
		flex-direction: column;
		gap: 2rem;
		align-items: center;
	}

	.stat-block {
		text-align: center;
	}

	.stat-label {
		display: block;
		font-size: 0.9rem;
		color: #888;
		text-transform: uppercase;
		letter-spacing: 0.1em;
		margin-bottom: 0.5rem;
	}

	.stat-value {
		font-size: 3rem;
		font-weight: 700;
		font-variant-numeric: tabular-nums;
	}

	.stat-value.gold {
		color: #ffd700;
	}

	.ledger-entry {
		width: 100%;
		max-width: 360px;
		background: #111;
		border-radius: 8px;
		padding: 1rem 1.5rem;
		font-family: 'SF Mono', 'Fira Code', monospace;
		font-size: 0.9rem;
	}

	.entry-row {
		display: grid;
		grid-template-columns: 1fr auto auto;
		gap: 1.5rem;
		padding: 0.4rem 0;
	}

	.entry-row.indent .account {
		padding-left: 1.5rem;
	}

	.account {
		color: #e8e8e8;
	}

	.debit {
		color: #4ade80;
		text-align: right;
	}

	.credit {
		color: #ffd700;
		text-align: right;
	}

	.balance-note {
		color: #555;
		font-size: 0.85rem;
		font-style: italic;
		text-align: center;
	}

	/* Chapter 5: Spoilage */
	.spoilage-total {
		text-align: center;
		margin-top: 0.75rem;
		color: #ef4444;
		font-size: 0.95rem;
	}

	/* Chapter 7: The Books */
	.books-panel {
		max-height: none;
	}

	.financial-summary,
	.trial-balance {
		margin-bottom: 2rem;
	}

	.section-title {
		font-size: 1.2rem;
		font-weight: 600;
		color: #ffd700;
		margin-bottom: 1rem;
		padding-bottom: 0.5rem;
		border-bottom: 1px solid #333;
	}

	.pl-section {
		margin-bottom: 1rem;
	}

	.pl-heading {
		font-size: 0.85rem;
		text-transform: uppercase;
		letter-spacing: 0.08em;
		margin-bottom: 0.5rem;
	}

	.revenue-heading {
		color: #4ade80;
	}

	.expense-heading {
		color: #ef4444;
	}

	.pl-row,
	.pl-total {
		display: flex;
		justify-content: space-between;
		padding: 0.3rem 0;
		font-size: 0.9rem;
	}

	.pl-account {
		color: #ccc;
	}

	.pl-amount {
		font-variant-numeric: tabular-nums;
		color: #e8e8e8;
	}

	.pl-total {
		border-top: 2px solid #444;
		margin-top: 0.5rem;
		padding-top: 0.75rem;
		font-weight: 700;
		font-size: 1rem;
	}

	.net-income {
		color: #4ade80;
		font-size: 1.1rem;
	}

	.tb-header,
	.tb-row,
	.tb-total {
		display: grid;
		grid-template-columns: 1fr auto auto;
		gap: 1rem;
		padding: 0.35rem 0;
		font-size: 0.85rem;
	}

	.tb-header {
		color: #888;
		text-transform: uppercase;
		letter-spacing: 0.05em;
		font-size: 0.75rem;
		border-bottom: 1px solid #333;
		padding-bottom: 0.5rem;
		margin-bottom: 0.25rem;
	}

	.tb-account {
		color: #ccc;
	}

	.tb-amount {
		font-variant-numeric: tabular-nums;
		text-align: right;
		color: #e8e8e8;
	}

	.tb-total {
		border-top: 2px solid #444;
		margin-top: 0.5rem;
		padding-top: 0.5rem;
		font-weight: 700;
	}

	.balance-status {
		text-align: center;
		margin-top: 1rem;
		font-size: 1rem;
		color: #ef4444;
		font-weight: 600;
	}

	.balance-status.balanced {
		color: #4ade80;
	}

	@media (max-width: 768px) {
		.chart-container {
			height: 220px;
		}

		.ledger-entry {
			font-size: 0.8rem;
		}
	}
</style>
