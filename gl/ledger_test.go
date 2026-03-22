package gl

import (
	"context"
	"testing"
	"time"

	fact "github.com/benaskins/axon-fact"
	"github.com/shopspring/decimal"
)

// mockAccounts is an in-memory chart of accounts for testing.
type mockAccounts struct {
	active map[string]AccountType
}

func (m *mockAccounts) Exists(_ context.Context, number string) (bool, error) {
	_, ok := m.active[number]
	return ok, nil
}

func (m *mockAccounts) accountType(number string) AccountType {
	return m.active[number]
}

func newTestLedger(accounts map[string]AccountType) (*Ledger, *BalanceProjection) {
	projection := NewBalanceProjection()
	store := fact.NewMemoryStore(fact.WithProjector(projection))
	mock := &mockAccounts{active: accounts}
	ledger := NewLedger(store, mock, "AUD")
	return ledger, projection
}

func TestPost_BalancedEntry(t *testing.T) {
	ledger, projection := newTestLedger(map[string]AccountType{
		"1000": Asset,
		"4000": Revenue,
	})
	ctx := context.Background()

	entryID, err := ledger.Post(ctx, JournalEntryPosted{
		Date:        time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC),
		Description: "Service revenue",
		Lines: []Line{
			{Account: "1000", Debit: decimal.NewFromInt(1000)},
			{Account: "4000", Credit: decimal.NewFromInt(1000)},
		},
	})
	if err != nil {
		t.Fatalf("Post: %v", err)
	}
	if entryID == "" {
		t.Fatal("expected non-empty entry ID")
	}

	// Check projection
	bal := projection.AccountBalance("1000")
	if !bal.Debits.Equal(decimal.NewFromInt(1000)) {
		t.Errorf("1000 debits = %s, want 1000", bal.Debits)
	}
	bal = projection.AccountBalance("4000")
	if !bal.Credits.Equal(decimal.NewFromInt(1000)) {
		t.Errorf("4000 credits = %s, want 1000", bal.Credits)
	}
}

func TestPost_UnbalancedEntry(t *testing.T) {
	ledger, _ := newTestLedger(map[string]AccountType{
		"1000": Asset,
		"4000": Revenue,
	})
	ctx := context.Background()

	_, err := ledger.Post(ctx, JournalEntryPosted{
		Date:        time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC),
		Description: "Bad entry",
		Lines: []Line{
			{Account: "1000", Debit: decimal.NewFromInt(1000)},
			{Account: "4000", Credit: decimal.NewFromInt(500)},
		},
	})
	if err == nil {
		t.Fatal("expected error for unbalanced entry")
	}
}

func TestPost_InactiveAccount(t *testing.T) {
	ledger, _ := newTestLedger(map[string]AccountType{
		"1000": Asset,
		// 4000 does not exist
	})
	ctx := context.Background()

	_, err := ledger.Post(ctx, JournalEntryPosted{
		Date:        time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC),
		Description: "Missing account",
		Lines: []Line{
			{Account: "1000", Debit: decimal.NewFromInt(100)},
			{Account: "4000", Credit: decimal.NewFromInt(100)},
		},
	})
	if err == nil {
		t.Fatal("expected error for nonexistent account")
	}
}

func TestPost_TooFewLines(t *testing.T) {
	ledger, _ := newTestLedger(map[string]AccountType{
		"1000": Asset,
	})
	ctx := context.Background()

	_, err := ledger.Post(ctx, JournalEntryPosted{
		Date:        time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC),
		Description: "Single line",
		Lines: []Line{
			{Account: "1000", Debit: decimal.NewFromInt(100)},
		},
	})
	if err == nil {
		t.Fatal("expected error for single-line entry")
	}
}

func TestPost_ZeroAmounts(t *testing.T) {
	ledger, _ := newTestLedger(map[string]AccountType{
		"1000": Asset,
		"2000": Liability,
	})
	ctx := context.Background()

	_, err := ledger.Post(ctx, JournalEntryPosted{
		Date:        time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC),
		Description: "Zero entry",
		Lines: []Line{
			{Account: "1000", Debit: decimal.Zero},
			{Account: "2000", Credit: decimal.Zero},
		},
	})
	if err == nil {
		t.Fatal("expected error for zero-amount entry")
	}
}

func TestTrialBalance(t *testing.T) {
	ledger, projection := newTestLedger(map[string]AccountType{
		"1000": Asset,
		"2000": Liability,
		"4000": Revenue,
		"5000": Expense,
	})
	ctx := context.Background()

	// Revenue received
	ledger.Post(ctx, JournalEntryPosted{
		Date:        time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC),
		Description: "Revenue",
		Lines: []Line{
			{Account: "1000", Debit: decimal.NewFromInt(5000)},
			{Account: "4000", Credit: decimal.NewFromInt(5000)},
		},
	})

	// Expense paid
	ledger.Post(ctx, JournalEntryPosted{
		Date:        time.Date(2026, 3, 5, 0, 0, 0, 0, time.UTC),
		Description: "Hosting",
		Lines: []Line{
			{Account: "5000", Debit: decimal.NewFromInt(200)},
			{Account: "1000", Credit: decimal.NewFromInt(200)},
		},
	})

	tb := projection.TrialBalance()
	if !tb.InBalance() {
		t.Errorf("trial balance not in balance: debits=%s credits=%s", tb.TotalDebits, tb.TotalCredits)
	}
	if !tb.TotalDebits.Equal(decimal.NewFromInt(5200)) {
		t.Errorf("total debits = %s, want 5200", tb.TotalDebits)
	}
}

func TestProfitAndLoss(t *testing.T) {
	accounts := map[string]AccountType{
		"1000": Asset,
		"4000": Revenue,
		"4100": Revenue,
		"5000": Expense,
	}
	ledger, projection := newTestLedger(accounts)
	mock := &mockAccounts{active: accounts}
	ctx := context.Background()

	// Two revenue entries
	ledger.Post(ctx, JournalEntryPosted{
		Date:        time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC),
		Description: "Consulting",
		Lines: []Line{
			{Account: "1000", Debit: decimal.NewFromInt(3000)},
			{Account: "4000", Credit: decimal.NewFromInt(3000)},
		},
	})
	ledger.Post(ctx, JournalEntryPosted{
		Date:        time.Date(2026, 3, 10, 0, 0, 0, 0, time.UTC),
		Description: "Product sales",
		Lines: []Line{
			{Account: "1000", Debit: decimal.NewFromInt(2000)},
			{Account: "4100", Credit: decimal.NewFromInt(2000)},
		},
	})

	// Expense
	ledger.Post(ctx, JournalEntryPosted{
		Date:        time.Date(2026, 3, 15, 0, 0, 0, 0, time.UTC),
		Description: "Hosting",
		Lines: []Line{
			{Account: "5000", Debit: decimal.NewFromInt(500)},
			{Account: "1000", Credit: decimal.NewFromInt(500)},
		},
	})

	from := time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2026, 3, 31, 23, 59, 59, 0, time.UTC)
	pl := projection.ProfitAndLoss(from, to, mock.accountType)

	if !pl.NetIncome.Equal(decimal.NewFromInt(4500)) {
		t.Errorf("net income = %s, want 4500", pl.NetIncome)
	}
	if len(pl.Revenue) != 2 {
		t.Errorf("revenue accounts = %d, want 2", len(pl.Revenue))
	}
	if len(pl.Expenses) != 1 {
		t.Errorf("expense accounts = %d, want 1", len(pl.Expenses))
	}
}

func TestPost_MultiCurrency(t *testing.T) {
	ledger, projection := newTestLedger(map[string]AccountType{
		"1000": Asset,
		"4000": Revenue,
	})
	ctx := context.Background()

	// USD invoice posted to AUD ledger at 1.5 exchange rate
	_, err := ledger.Post(ctx, JournalEntryPosted{
		Date:        time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC),
		Description: "USD invoice",
		Lines: []Line{
			{Account: "1000", Debit: decimal.NewFromInt(1000), Currency: "USD", ExchangeRate: decimal.NewFromFloat(1.5)},
			{Account: "4000", Credit: decimal.NewFromInt(1000), Currency: "USD", ExchangeRate: decimal.NewFromFloat(1.5)},
		},
	})
	if err != nil {
		t.Fatalf("Post: %v", err)
	}

	// Balance should be in base currency (AUD)
	bal := projection.AccountBalance("1000")
	if !bal.Debits.Equal(decimal.NewFromInt(1500)) {
		t.Errorf("1000 debits = %s, want 1500 (1000 USD * 1.5)", bal.Debits)
	}
}

func TestPost_SourceLabelling(t *testing.T) {
	ledger, _ := newTestLedger(map[string]AccountType{
		"1100": Asset,
		"4000": Revenue,
	})
	ctx := context.Background()

	entryID, err := ledger.Post(ctx, JournalEntryPosted{
		Date:        time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC),
		Description: "Invoice INV-001",
		SourceType:  "invoice",
		SourceRef:   "inv-001",
		Kind:        Operating,
		Lines: []Line{
			{Account: "1100", Debit: decimal.NewFromInt(500)},
			{Account: "4000", Credit: decimal.NewFromInt(500)},
		},
	})
	if err != nil {
		t.Fatalf("Post: %v", err)
	}
	if entryID == "" {
		t.Fatal("expected entry ID")
	}
}

func TestPost_DefaultKind(t *testing.T) {
	ledger, _ := newTestLedger(map[string]AccountType{
		"1000": Asset,
		"4000": Revenue,
	})
	ctx := context.Background()

	// Post without setting Kind — should default to Operating
	_, err := ledger.Post(ctx, JournalEntryPosted{
		Date:        time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC),
		Description: "Default kind",
		Lines: []Line{
			{Account: "1000", Debit: decimal.NewFromInt(100)},
			{Account: "4000", Credit: decimal.NewFromInt(100)},
		},
	})
	if err != nil {
		t.Fatalf("Post: %v", err)
	}
}
