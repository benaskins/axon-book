package gl

import (
	"time"

	"github.com/shopspring/decimal"
)

// EntryKind classifies a journal entry by its accounting purpose.
type EntryKind string

const (
	Operating EntryKind = "operating"
	Adjusting EntryKind = "adjusting"
	Closing   EntryKind = "closing"
	Reversing EntryKind = "reversing"
)

// Event type constants for journal entry events.
const (
	EventJournalEntryPosted = "journal_entry.posted"
)

// JournalEntryPosted is the event data for a posted journal entry.
type JournalEntryPosted struct {
	EntryID       string    `json:"entry_id"`
	Date          time.Time `json:"date"`
	Description   string    `json:"description"`
	Lines         []Line    `json:"lines"`
	SourceType    string    `json:"source_type,omitempty"`
	SourceRef     string    `json:"source_ref,omitempty"`
	Kind          EntryKind `json:"kind"`
	ReversesEntry string    `json:"reverses_entry,omitempty"`
}

// Line is a single debit or credit within a journal entry.
type Line struct {
	Account      string          `json:"account"`
	Debit        decimal.Decimal `json:"debit"`
	Credit       decimal.Decimal `json:"credit"`
	Currency     string          `json:"currency,omitempty"`
	ExchangeRate decimal.Decimal `json:"exchange_rate,omitempty"`
	Description  string          `json:"description,omitempty"`
}

// BaseAmount returns the line amount converted to the ledger's base currency.
// If no exchange rate is set, the original amount is returned.
func (l Line) BaseAmount() (debit, credit decimal.Decimal) {
	if l.ExchangeRate.IsZero() {
		return l.Debit, l.Credit
	}
	return l.Debit.Mul(l.ExchangeRate), l.Credit.Mul(l.ExchangeRate)
}
