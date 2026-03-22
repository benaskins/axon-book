export interface JournalLine {
	account: string;
	debit: string;
	credit: string;
	description?: string;
	currency?: string;
	exchange_rate?: string;
}

export interface JournalEntry {
	entry_id: string;
	date: string;
	description: string;
	kind: string;
	lines: JournalLine[];
	source_type?: string;
	source_ref?: string;
}

export interface DomainEvent {
	id: string;
	type: string;
	data: Record<string, unknown>;
	metadata?: Record<string, string>;
	sequence: number;
	occurred_at: string;
	journal_entry?: JournalEntry;
}

export interface TAccountEntry {
	eventIndex: number;
	amount: number;
	side: 'debit' | 'credit';
	description: string;
}
