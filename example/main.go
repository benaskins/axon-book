// Example demonstrates composing a general ledger from axon-book/gl:
// set up a chart of accounts, post journal entries, and query trial balance and P&L.
//
// This example uses axon-fact's MemoryStore for simplicity.
// A production composition root would use fact.PostgresStore with axon.OpenDB
// and run both fact.Migrations and gl.Migrations via axon.RunMigrations.
package main

import (
	"context"
	"fmt"
	"log"
	"time"

	fact "github.com/benaskins/axon-fact"
	"github.com/benaskins/axon-book/gl"
	"github.com/shopspring/decimal"
)

// inMemoryAccounts is a simple in-memory chart of accounts for the example.
type inMemoryAccounts struct {
	accounts map[string]gl.AccountType
}

func (m *inMemoryAccounts) Exists(_ context.Context, number string) (bool, error) {
	_, ok := m.accounts[number]
	return ok, nil
}

func (m *inMemoryAccounts) accountType(number string) gl.AccountType {
	return m.accounts[number]
}

func main() {
	ctx := context.Background()

	// --- Chart of accounts ---
	accounts := &inMemoryAccounts{accounts: map[string]gl.AccountType{
		"1000": gl.Asset,   // Cash
		"1100": gl.Asset,   // Accounts Receivable
		"2000": gl.Liability, // Accounts Payable
		"3000": gl.Equity,  // Owner's Equity
		"4000": gl.Revenue, // Consulting Revenue
		"4100": gl.Revenue, // Product Revenue
		"5000": gl.Expense, // Hosting
		"5100": gl.Expense, // Software Subscriptions
	}}

	// --- Event store + projection ---
	projection := gl.NewBalanceProjection()
	store := fact.NewMemoryStore(fact.WithProjector(projection))

	// --- Ledger ---
	ledger := gl.NewLedger(store, accounts, "AUD")
	fmt.Printf("Ledger base currency: %s\n\n", ledger.BaseCurrency())

	// --- Post journal entries ---

	// 1. Consulting revenue received in cash
	ledger.Post(ctx, gl.JournalEntryPosted{
		Date:        time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC),
		Description: "Consulting engagement - March",
		Kind:        gl.Operating,
		Lines: []gl.Line{
			{Account: "1000", Debit: decimal.NewFromInt(5000)},
			{Account: "4000", Credit: decimal.NewFromInt(5000)},
		},
	})

	// 2. Product sale on credit
	if _, err := ledger.Post(ctx, gl.JournalEntryPosted{
		Date:        time.Date(2026, 3, 5, 0, 0, 0, 0, time.UTC),
		Description: "Product sale INV-001",
		Kind:        gl.Operating,
		SourceType:  "invoice",
		SourceRef:   "inv-001",
		Lines: []gl.Line{
			{Account: "1100", Debit: decimal.NewFromInt(2000)},
			{Account: "4100", Credit: decimal.NewFromInt(2000)},
		},
	}); err != nil {
		log.Fatal(err)
	}

	// 3. Pay hosting bill
	ledger.Post(ctx, gl.JournalEntryPosted{
		Date:        time.Date(2026, 3, 10, 0, 0, 0, 0, time.UTC),
		Description: "OrbStack hosting - March",
		Kind:        gl.Operating,
		Lines: []gl.Line{
			{Account: "5000", Debit: decimal.NewFromInt(200)},
			{Account: "1000", Credit: decimal.NewFromInt(200)},
		},
	})

	// 4. USD software subscription (multi-currency)
	ledger.Post(ctx, gl.JournalEntryPosted{
		Date:        time.Date(2026, 3, 15, 0, 0, 0, 0, time.UTC),
		Description: "GitHub subscription (USD)",
		Kind:        gl.Operating,
		Lines: []gl.Line{
			{Account: "5100", Debit: decimal.NewFromInt(4), Currency: "USD", ExchangeRate: decimal.NewFromFloat(1.55)},
			{Account: "1000", Credit: decimal.NewFromInt(4), Currency: "USD", ExchangeRate: decimal.NewFromFloat(1.55)},
		},
	})

	// --- Reports ---

	// Trial balance
	fmt.Println("=== Trial Balance ===")
	tb := projection.TrialBalance()
	for _, bal := range tb.Balances {
		name := accounts.accounts[bal.Account]
		fmt.Printf("  %-6s %-12s  DR: %8s  CR: %8s\n",
			bal.Account, name, bal.Debits, bal.Credits)
	}
	fmt.Printf("  %-20s  DR: %8s  CR: %8s\n", "TOTALS", tb.TotalDebits, tb.TotalCredits)
	fmt.Printf("  In balance: %v\n\n", tb.InBalance())

	// Profit & Loss
	fmt.Println("=== Profit & Loss (March 2026) ===")
	from := time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2026, 3, 31, 23, 59, 59, 0, time.UTC)
	pl := projection.ProfitAndLoss(from, to, accounts.accountType)

	fmt.Println("  Revenue:")
	for _, r := range pl.Revenue {
		fmt.Printf("    %-6s  %s\n", r.Account, r.Credits.Sub(r.Debits))
	}
	fmt.Println("  Expenses:")
	for _, e := range pl.Expenses {
		fmt.Printf("    %-6s  %s\n", e.Account, e.Debits.Sub(e.Credits))
	}
	fmt.Printf("  Net Income: %s\n", pl.NetIncome)
}
