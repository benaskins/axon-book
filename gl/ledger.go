package gl

import (
	"context"
	"encoding/json"
	"fmt"

	fact "github.com/benaskins/axon-fact"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// AccountChecker validates that an account exists and is active.
type AccountChecker interface {
	Exists(ctx context.Context, number string) (bool, error)
}

// Ledger is the general ledger. It validates and posts journal entries
// to an event store, with projections kept in sync via projectors.
type Ledger struct {
	events   fact.EventStore
	accounts AccountChecker
	currency string // base currency (ISO 4217)
}

// NewLedger creates a general ledger with the given base currency.
func NewLedger(events fact.EventStore, accounts AccountChecker, baseCurrency string) *Ledger {
	return &Ledger{
		events:   events,
		accounts: accounts,
		currency: baseCurrency,
	}
}

// BaseCurrency returns the ledger's base currency code.
func (l *Ledger) BaseCurrency() string {
	return l.currency
}

// Post validates and appends a journal entry to the event store.
// It enforces: at least two lines, all accounts exist and are active,
// and total base-currency debits equal total base-currency credits.
func (l *Ledger) Post(ctx context.Context, entry JournalEntryPosted) (string, error) {
	return l.PostWithMetadata(ctx, entry, nil)
}

// PostWithMetadata is like Post but attaches metadata to the event.
// Use this to link journal entries to their causing domain events
// via a causation_id.
func (l *Ledger) PostWithMetadata(ctx context.Context, entry JournalEntryPosted, metadata map[string]string) (string, error) {
	if len(entry.Lines) < 2 {
		return "", fmt.Errorf("journal entry must have at least two lines")
	}

	if entry.Kind == "" {
		entry.Kind = Operating
	}

	// Validate accounts exist and are active
	for _, line := range entry.Lines {
		exists, err := l.accounts.Exists(ctx, line.Account)
		if err != nil {
			return "", fmt.Errorf("check account %s: %w", line.Account, err)
		}
		if !exists {
			return "", fmt.Errorf("account %s does not exist or is inactive", line.Account)
		}
	}

	// Validate debits == credits in base currency
	totalDebits := decimal.Zero
	totalCredits := decimal.Zero
	for _, line := range entry.Lines {
		d, c := line.BaseAmount()
		totalDebits = totalDebits.Add(d)
		totalCredits = totalCredits.Add(c)
	}
	if !totalDebits.Equal(totalCredits) {
		return "", fmt.Errorf("entry does not balance: debits=%s credits=%s", totalDebits, totalCredits)
	}

	if totalDebits.IsZero() {
		return "", fmt.Errorf("journal entry must have non-zero amounts")
	}

	// Assign entry ID if not set
	if entry.EntryID == "" {
		entry.EntryID = uuid.New().String()
	}

	data, err := json.Marshal(entry)
	if err != nil {
		return "", fmt.Errorf("marshal entry: %w", err)
	}

	event := fact.Event{
		ID:       uuid.New().String(),
		Type:     EventJournalEntryPosted,
		Data:     data,
		Metadata: metadata,
	}

	if err := l.events.Append(ctx, StreamLedger, []fact.Event{event}); err != nil {
		return "", fmt.Errorf("append journal entry: %w", err)
	}

	return entry.EntryID, nil
}
