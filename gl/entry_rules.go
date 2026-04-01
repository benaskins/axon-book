package gl

import (
	"strings"

	"github.com/benaskins/axon-rule"
	"github.com/shopspring/decimal"
)

// ViolationError wraps rule.Violations as an error so callers can access
// structured violations via type assertion.
type ViolationError struct {
	Violations rule.Violations
}

func (e *ViolationError) Error() string {
	codes := e.Violations.Codes()
	return "validation failed: " + strings.Join(codes, ", ")
}

// Violation context types for journal entry business rules.
type TooFewLines struct{}
type ZeroAmounts struct{}

// BalanceMismatch carries the totals when debits don't equal credits.
type BalanceMismatch struct {
	TotalDebits  string
	TotalCredits string
}

// JournalEntryIsValid defines the business rules for posting a journal entry.
// Account existence is validated separately as it requires I/O.
var JournalEntryIsValid = rule.AllOf(
	rule.New(JournalEntryPosted.HasAtLeastTwoLines),
	rule.New(JournalEntryPosted.DebitsEqualCredits),
	rule.New(JournalEntryPosted.HasNonZeroAmounts),
)

// HasAtLeastTwoLines checks the entry has two or more lines.
func (e JournalEntryPosted) HasAtLeastTwoLines() rule.Verdict {
	if len(e.Lines) >= 2 {
		return rule.Pass()
	}
	return rule.FailWith(TooFewLines{})
}

// DebitsEqualCredits checks total base-currency debits equal credits.
func (e JournalEntryPosted) DebitsEqualCredits() rule.Verdict {
	totalDebits := decimal.Zero
	totalCredits := decimal.Zero
	for _, line := range e.Lines {
		d, c := line.BaseAmount()
		totalDebits = totalDebits.Add(d)
		totalCredits = totalCredits.Add(c)
	}
	if totalDebits.Equal(totalCredits) {
		return rule.Pass()
	}
	return rule.FailWith(BalanceMismatch{
		TotalDebits:  totalDebits.String(),
		TotalCredits: totalCredits.String(),
	})
}

// HasNonZeroAmounts checks the entry has non-zero total amounts.
func (e JournalEntryPosted) HasNonZeroAmounts() rule.Verdict {
	total := decimal.Zero
	for _, line := range e.Lines {
		d, c := line.BaseAmount()
		total = total.Add(d).Add(c)
	}
	if !total.IsZero() {
		return rule.Pass()
	}
	return rule.FailWith(ZeroAmounts{})
}
