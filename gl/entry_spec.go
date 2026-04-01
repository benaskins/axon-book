package gl

import (
	"strings"

	spec "github.com/benaskins/axon-spec"
	"github.com/shopspring/decimal"
)

// ViolationError wraps a spec.Result as an error so callers can access
// structured violations via type assertion.
type ViolationError struct {
	Result spec.Result
}

func (e *ViolationError) Error() string {
	codes := e.Result.ViolationCodes()
	strs := make([]string, len(codes))
	for i, c := range codes {
		strs[i] = string(c)
	}
	return "validation failed: " + strings.Join(strs, ", ")
}

// Violation codes for journal entry business rules.
const (
	MustHaveAtLeastTwoLines spec.Code = "must-have-at-least-two-lines"
	DebitsMustEqualCredits  spec.Code = "debits-must-equal-credits"
	MustHaveNonZeroAmounts  spec.Code = "must-have-non-zero-amounts"
)

// JournalEntryIsValid defines the business rules for posting a journal entry.
// Account existence is validated separately as it requires I/O.
var JournalEntryIsValid = spec.AllOf(
	spec.New(MustHaveAtLeastTwoLines, JournalEntryPosted.HasAtLeastTwoLines),
	spec.New(DebitsMustEqualCredits, JournalEntryPosted.DebitsEqualCredits),
	spec.New(MustHaveNonZeroAmounts, JournalEntryPosted.HasNonZeroAmounts),
)

// HasAtLeastTwoLines checks the entry has two or more lines.
func (e JournalEntryPosted) HasAtLeastTwoLines() spec.PredicateResult {
	if len(e.Lines) >= 2 {
		return spec.Pass()
	}
	return spec.Fail()
}

// DebitsEqualCredits checks total base-currency debits equal credits.
func (e JournalEntryPosted) DebitsEqualCredits() spec.PredicateResult {
	totalDebits := decimal.Zero
	totalCredits := decimal.Zero
	for _, line := range e.Lines {
		d, c := line.BaseAmount()
		totalDebits = totalDebits.Add(d)
		totalCredits = totalCredits.Add(c)
	}
	if totalDebits.Equal(totalCredits) {
		return spec.Pass()
	}
	return spec.FailWith(map[string]any{
		"total_debits":  totalDebits.String(),
		"total_credits": totalCredits.String(),
	})
}

// HasNonZeroAmounts checks the entry has non-zero total amounts.
func (e JournalEntryPosted) HasNonZeroAmounts() spec.PredicateResult {
	total := decimal.Zero
	for _, line := range e.Lines {
		d, c := line.BaseAmount()
		total = total.Add(d).Add(c)
	}
	if !total.IsZero() {
		return spec.Pass()
	}
	return spec.Fail()
}
