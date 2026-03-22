package gl

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	fact "github.com/benaskins/axon-fact"
	"github.com/shopspring/decimal"
)

// Balance holds the accumulated debits and credits for an account.
type Balance struct {
	Account string          `json:"account"`
	Debits  decimal.Decimal `json:"debits"`
	Credits decimal.Decimal `json:"credits"`
}

// Net returns debits minus credits.
func (b Balance) Net() decimal.Decimal {
	return b.Debits.Sub(b.Credits)
}

// TrialBalance is a snapshot of all account balances.
type TrialBalance struct {
	Balances     []Balance       `json:"balances"`
	TotalDebits  decimal.Decimal `json:"total_debits"`
	TotalCredits decimal.Decimal `json:"total_credits"`
}

// InBalance returns true if total debits equal total credits.
func (tb TrialBalance) InBalance() bool {
	return tb.TotalDebits.Equal(tb.TotalCredits)
}

// ProfitAndLoss shows revenue and expense totals over a period.
type ProfitAndLoss struct {
	From     time.Time       `json:"from"`
	To       time.Time       `json:"to"`
	Revenue  []Balance       `json:"revenue"`
	Expenses []Balance       `json:"expenses"`
	NetIncome decimal.Decimal `json:"net_income"`
}

// BalanceProjection builds account balances from journal entry events.
// It implements fact.Projector and runs synchronously within Append.
type BalanceProjection struct {
	mu       sync.RWMutex
	balances map[string]*Balance
	entries  []journalRecord // for date-range queries
}

type journalRecord struct {
	entry JournalEntryPosted
	date  time.Time
}

// NewBalanceProjection creates an empty balance projection.
func NewBalanceProjection() *BalanceProjection {
	return &BalanceProjection{
		balances: make(map[string]*Balance),
	}
}

// Handle processes a single event to update account balances.
func (p *BalanceProjection) Handle(_ context.Context, event fact.Event) error {
	if event.Type != EventJournalEntryPosted {
		return nil
	}

	var entry JournalEntryPosted
	if err := json.Unmarshal(event.Data, &entry); err != nil {
		return fmt.Errorf("unmarshal journal entry: %w", err)
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	p.entries = append(p.entries, journalRecord{entry: entry, date: entry.Date})

	for _, line := range entry.Lines {
		bal, ok := p.balances[line.Account]
		if !ok {
			bal = &Balance{Account: line.Account}
			p.balances[line.Account] = bal
		}
		d, c := line.BaseAmount()
		bal.Debits = bal.Debits.Add(d)
		bal.Credits = bal.Credits.Add(c)
	}

	return nil
}

// TrialBalance returns the current trial balance across all accounts.
func (p *BalanceProjection) TrialBalance() TrialBalance {
	p.mu.RLock()
	defer p.mu.RUnlock()

	var tb TrialBalance
	for _, bal := range p.balances {
		tb.Balances = append(tb.Balances, *bal)
		tb.TotalDebits = tb.TotalDebits.Add(bal.Debits)
		tb.TotalCredits = tb.TotalCredits.Add(bal.Credits)
	}
	return tb
}

// AccountBalance returns the balance for a single account.
func (p *BalanceProjection) AccountBalance(account string) Balance {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if bal, ok := p.balances[account]; ok {
		return *bal
	}
	return Balance{Account: account}
}

// ProfitAndLoss computes P&L for a date range using the given account type resolver.
// The resolver maps account numbers to their AccountType.
func (p *BalanceProjection) ProfitAndLoss(from, to time.Time, accountType func(string) AccountType) ProfitAndLoss {
	p.mu.RLock()
	defer p.mu.RUnlock()

	// Accumulate balances for the period only
	periodBalances := make(map[string]*Balance)
	for _, rec := range p.entries {
		if rec.date.Before(from) || rec.date.After(to) {
			continue
		}
		for _, line := range rec.entry.Lines {
			bal, ok := periodBalances[line.Account]
			if !ok {
				bal = &Balance{Account: line.Account}
				periodBalances[line.Account] = bal
			}
			d, c := line.BaseAmount()
			bal.Debits = bal.Debits.Add(d)
			bal.Credits = bal.Credits.Add(c)
		}
	}

	pl := ProfitAndLoss{From: from, To: to}
	totalRevenue := decimal.Zero
	totalExpenses := decimal.Zero

	for _, bal := range periodBalances {
		switch accountType(bal.Account) {
		case Revenue:
			pl.Revenue = append(pl.Revenue, *bal)
			// Revenue has a natural credit balance
			totalRevenue = totalRevenue.Add(bal.Credits.Sub(bal.Debits))
		case Expense:
			pl.Expenses = append(pl.Expenses, *bal)
			// Expenses have a natural debit balance
			totalExpenses = totalExpenses.Add(bal.Debits.Sub(bal.Credits))
		}
	}

	pl.NetIncome = totalRevenue.Sub(totalExpenses)
	return pl
}

// Verify BalanceProjection satisfies Projector at compile time.
var _ fact.Projector = (*BalanceProjection)(nil)
