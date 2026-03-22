export interface DailySummary {
	date: string;
	cups_sold: number;
	price_per_cup: string;
	revenue: string;
	weather: string;
	cogs_lemons: string;
	cogs_sugar: string;
	cogs_cups: string;
	cogs_ice: string;
	spoilage: string;
	advertising: string;
	permit: string;
	ice_cost: string;
}

export interface MonthlySummary {
	month: string;
	revenue: string;
	total_cogs: string;
	spoilage: string;
	advertising: string;
	permit: string;
	ice_cost: string;
	cups_sold: number;
}

export interface Account {
	number: string;
	name: string;
	type: string;
	parent: string;
	active: boolean;
	created_at: string;
}

export interface TrialBalanceEntry {
	account: string;
	debits: string;
	credits: string;
	net: string;
}

export interface TrialBalance {
	balances: TrialBalanceEntry[];
	total_debits: string;
	total_credits: string;
	in_balance: boolean;
}

export interface ProfitAndLossEntry {
	account: string;
	debits: string;
	credits: string;
	net: string;
}

export interface ProfitAndLoss {
	from: string;
	to: string;
	revenue: ProfitAndLossEntry[];
	expenses: ProfitAndLossEntry[];
	net_income: string;
}

export async function fetchDailySummaries(from: string, to: string): Promise<DailySummary[]> {
	const res = await fetch(`/api/daily-summaries?from=${from}&to=${to}`);
	if (!res.ok) throw new Error(`Failed to fetch daily summaries: ${res.statusText}`);
	return res.json();
}

export async function fetchMonthlySummary(): Promise<MonthlySummary[]> {
	const res = await fetch('/api/monthly-summary');
	if (!res.ok) throw new Error(`Failed to fetch monthly summary: ${res.statusText}`);
	return res.json();
}

export async function fetchAccounts(): Promise<Account[]> {
	const res = await fetch('/api/accounts');
	if (!res.ok) throw new Error(`Failed to fetch accounts: ${res.statusText}`);
	return res.json();
}

export async function fetchTrialBalance(): Promise<TrialBalance> {
	const res = await fetch('/api/trial-balance');
	if (!res.ok) throw new Error(`Failed to fetch trial balance: ${res.statusText}`);
	return res.json();
}

export async function fetchProfitAndLoss(from: string, to: string): Promise<ProfitAndLoss> {
	const res = await fetch(`/api/profit-and-loss?from=${from}&to=${to}`);
	if (!res.ok) throw new Error(`Failed to fetch P&L: ${res.statusText}`);
	return res.json();
}
