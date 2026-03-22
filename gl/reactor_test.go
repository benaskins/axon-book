package gl

import (
	"context"
	"encoding/json"
	"testing"

	fact "github.com/benaskins/axon-fact"
	"github.com/shopspring/decimal"
)

// lemonadeAccounts returns the full chart of accounts for the lemonade stand.
func lemonadeAccounts() map[string]AccountType {
	return map[string]AccountType{
		"1000": Asset,   // Cash
		"1100": Asset,   // Inventory - Lemons
		"1200": Asset,   // Inventory - Sugar
		"1300": Asset,   // Inventory - Cups
		"1400": Asset,   // Inventory - Ice
		"3000": Equity,  // Owner's Equity
		"3100": Equity,  // Retained Earnings
		"4000": Revenue, // Lemonade Sales
		"5000": Expense, // COGS - Lemons
		"5100": Expense, // COGS - Sugar
		"5200": Expense, // COGS - Cups
		"5300": Expense, // COGS - Ice
		"5400": Expense, // Advertising
		"5500": Expense, // Stand Permit
		"5600": Expense, // Spoilage
	}
}

// newReactorTestLedger creates a store with both the reactor and balance projection wired up.
func newReactorTestLedger() (*fact.MemoryStore, *Ledger, *BalanceProjection) {
	accounts := &mockAccounts{active: lemonadeAccounts()}
	projection := NewBalanceProjection()

	// We need to create the store first, then the ledger, then wire the reactor.
	// The reactor needs the ledger, and the ledger needs the store.
	// We register the reactor as a projector on the store.
	var store *fact.MemoryStore
	var ledger *Ledger

	reactor := &deferredReactor{}
	store = fact.NewMemoryStore(
		fact.WithProjector(reactor),
		fact.WithProjector(projection),
	)
	ledger = NewLedger(store, accounts, "AUD")
	reactor.inner = NewReactor(ledger)

	return store, ledger, projection
}

// deferredReactor wraps Reactor to break the circular init dependency.
type deferredReactor struct {
	inner *Reactor
}

func (d *deferredReactor) Handle(ctx context.Context, e fact.Event) error {
	return d.inner.Handle(ctx, e)
}

func TestReactor_InvestmentMade(t *testing.T) {
	store, _, projection := newReactorTestLedger()
	ctx := context.Background()

	data, _ := json.Marshal(InvestmentMade{
		Date:        "2025-01-01",
		Amount:      decimal.NewFromInt(5000),
		Description: "Owner's initial investment",
	})

	err := store.Append(ctx, StreamOperations, []fact.Event{
		{ID: "inv-1", Type: EventInvestmentMade, Data: data},
	})
	if err != nil {
		t.Fatalf("Append: %v", err)
	}

	// Check the balance projection was updated via the reactor
	cash := projection.AccountBalance("1000")
	if !cash.Debits.Equal(decimal.NewFromInt(5000)) {
		t.Errorf("cash debits = %s, want 5000", cash.Debits)
	}

	equity := projection.AccountBalance("3000")
	if !equity.Credits.Equal(decimal.NewFromInt(5000)) {
		t.Errorf("equity credits = %s, want 5000", equity.Credits)
	}

	// Verify ledger stream has the journal entry
	ledgerEvents, _ := store.Load(ctx, StreamLedger)
	if len(ledgerEvents) != 1 {
		t.Fatalf("ledger events = %d, want 1", len(ledgerEvents))
	}
	if ledgerEvents[0].Type != EventJournalEntryPosted {
		t.Errorf("ledger event type = %q", ledgerEvents[0].Type)
	}
	if ledgerEvents[0].Metadata["causation_id"] != "inv-1" {
		t.Errorf("causation_id = %q, want %q", ledgerEvents[0].Metadata["causation_id"], "inv-1")
	}
}

func TestReactor_SaleCompleted(t *testing.T) {
	store, _, projection := newReactorTestLedger()
	ctx := context.Background()

	// First invest so there's cash
	investData, _ := json.Marshal(InvestmentMade{
		Date:   "2025-01-01",
		Amount: decimal.NewFromInt(5000),
	})
	store.Append(ctx, StreamOperations, []fact.Event{
		{ID: "inv-1", Type: EventInvestmentMade, Data: investData},
	})

	// Sale of 10 cups at $3.50
	saleData, _ := json.Marshal(SaleCompleted{
		Date:        "2025-06-15",
		Cups:        10,
		PricePerCup: decimal.NewFromFloat(3.50),
		Weather:     "hot",
	})

	err := store.Append(ctx, StreamOperations, []fact.Event{
		{ID: "sale-1", Type: EventSaleCompleted, Data: saleData},
	})
	if err != nil {
		t.Fatalf("Append: %v", err)
	}

	// Revenue: 10 * 3.50 = 35.00
	revenue := projection.AccountBalance("4000")
	if !revenue.Credits.Equal(decimal.NewFromFloat(35.00)) {
		t.Errorf("revenue credits = %s, want 35.00", revenue.Credits)
	}

	// COGS Lemons: 10 * 0.50 = 5.00
	cogsLemons := projection.AccountBalance("5000")
	if !cogsLemons.Debits.Equal(decimal.NewFromFloat(5.00)) {
		t.Errorf("COGS lemons = %s, want 5.00", cogsLemons.Debits)
	}

	// Verify causation link
	ledgerEvents, _ := store.Load(ctx, StreamLedger)
	// Event 0 is investment, event 1 is sale
	if len(ledgerEvents) < 2 {
		t.Fatalf("ledger events = %d, want >= 2", len(ledgerEvents))
	}
	if ledgerEvents[1].Metadata["causation_id"] != "sale-1" {
		t.Errorf("causation_id = %q, want %q", ledgerEvents[1].Metadata["causation_id"], "sale-1")
	}
}

func TestReactor_SupplyPurchased(t *testing.T) {
	store, _, projection := newReactorTestLedger()
	ctx := context.Background()

	investData, _ := json.Marshal(InvestmentMade{Date: "2025-01-01", Amount: decimal.NewFromInt(5000)})
	store.Append(ctx, StreamOperations, []fact.Event{
		{ID: "inv-1", Type: EventInvestmentMade, Data: investData},
	})

	supplyData, _ := json.Marshal(SupplyPurchased{
		Date: "2025-03-03",
		Ref:  "po-20250303",
		Items: []SupplyItem{
			{Name: "Lemons", Account: "1100", Quantity: 30, Cost: decimal.NewFromInt(30)},
			{Name: "Sugar", Account: "1200", Quantity: 15, Cost: decimal.NewFromInt(15)},
			{Name: "Cups", Account: "1300", Quantity: 100, Cost: decimal.NewFromInt(10)},
		},
	})

	err := store.Append(ctx, StreamOperations, []fact.Event{
		{ID: "supply-1", Type: EventSupplyPurchased, Data: supplyData},
	})
	if err != nil {
		t.Fatalf("Append: %v", err)
	}

	lemons := projection.AccountBalance("1100")
	if !lemons.Debits.Equal(decimal.NewFromInt(30)) {
		t.Errorf("lemons inventory = %s, want 30", lemons.Debits)
	}

	// Cash should be reduced by 55 (total supplies)
	cash := projection.AccountBalance("1000")
	expectedCashCredits := decimal.NewFromInt(55)
	if !cash.Credits.Equal(expectedCashCredits) {
		t.Errorf("cash credits = %s, want %s", cash.Credits, expectedCashCredits)
	}
}

func TestReactor_SpoilageRecorded(t *testing.T) {
	store, _, projection := newReactorTestLedger()
	ctx := context.Background()

	spoilData, _ := json.Marshal(SpoilageRecorded{
		Date:    "2025-06-20",
		Item:    "lemons",
		Account: "1100",
		Amount:  decimal.NewFromInt(12),
		Reason:  "overripe",
	})

	err := store.Append(ctx, StreamOperations, []fact.Event{
		{ID: "spoil-1", Type: EventSpoilageRecorded, Data: spoilData},
	})
	if err != nil {
		t.Fatalf("Append: %v", err)
	}

	spoilage := projection.AccountBalance("5600")
	if !spoilage.Debits.Equal(decimal.NewFromInt(12)) {
		t.Errorf("spoilage = %s, want 12", spoilage.Debits)
	}
}

func TestReactor_IgnoresLedgerEvents(t *testing.T) {
	store, _, _ := newReactorTestLedger()
	ctx := context.Background()

	// Append a domain event to trigger the reactor
	investData, _ := json.Marshal(InvestmentMade{
		Date:   "2025-01-01",
		Amount: decimal.NewFromInt(100),
	})
	err := store.Append(ctx, StreamOperations, []fact.Event{
		{ID: "inv-1", Type: EventInvestmentMade, Data: investData},
	})
	if err != nil {
		t.Fatalf("Append: %v", err)
	}

	// Should have 1 operations event and 1 ledger event
	ops, _ := store.Load(ctx, StreamOperations)
	ledger, _ := store.Load(ctx, StreamLedger)

	if len(ops) != 1 {
		t.Errorf("operations events = %d, want 1", len(ops))
	}
	if len(ledger) != 1 {
		t.Errorf("ledger events = %d, want 1 (reactor should not recurse)", len(ledger))
	}
}
