// Chart of accounts for the lemonade stand.
// Static — matches the seed data.

export interface Account {
	number: string;
	name: string;
	type: 'asset' | 'equity' | 'revenue' | 'expense';
}

export const accounts: Record<string, Account> = {
	'1000': { number: '1000', name: 'Cash', type: 'asset' },
	'1100': { number: '1100', name: 'Lemons', type: 'asset' },
	'1200': { number: '1200', name: 'Sugar', type: 'asset' },
	'1300': { number: '1300', name: 'Cups', type: 'asset' },
	'1400': { number: '1400', name: 'Ice', type: 'asset' },
	'3000': { number: '3000', name: "Owner's Equity", type: 'equity' },
	'3100': { number: '3100', name: 'Retained Earnings', type: 'equity' },
	'4000': { number: '4000', name: 'Sales', type: 'revenue' },
	'5000': { number: '5000', name: 'COGS Lemons', type: 'expense' },
	'5100': { number: '5100', name: 'COGS Sugar', type: 'expense' },
	'5200': { number: '5200', name: 'COGS Cups', type: 'expense' },
	'5300': { number: '5300', name: 'COGS Ice', type: 'expense' },
	'5400': { number: '5400', name: 'Advertising', type: 'expense' },
	'5500': { number: '5500', name: 'Permit', type: 'expense' },
	'5600': { number: '5600', name: 'Spoilage', type: 'expense' },
};

export const accountGroups = [
	{ label: 'Assets', type: 'asset' as const },
	{ label: 'Equity', type: 'equity' as const },
	{ label: 'Revenue', type: 'revenue' as const },
	{ label: 'Expenses', type: 'expense' as const },
];
