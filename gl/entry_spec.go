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

// HasAtLeastTwoLines returns true when the entry has two or more lines.
func (e JournalEntryPosted) HasAtLeastTwoLines() (bool, map[string]any) {
	return len(e.Lines) >= 2, nil
}

// DebitsEqualCredits returns true when total base-currency debits equal credits.
func (e JournalEntryPosted) DebitsEqualCredits() (bool, map[string]any) {
	totalDebits := decimal.Zero
	totalCredits := decimal.Zero
	for _, line := range e.Lines {
		d, c := line.BaseAmount()
		totalDebits = totalDebits.Add(d)
		totalCredits = totalCredits.Add(c)
	}
	ok := totalDebits.Equal(totalCredits)
	return ok, map[string]any{
		"total_debits":  totalDebits.String(),
		"total_credits": totalCredits.String(),
	}
}

// HasNonZeroAmounts returns true when the entry has non-zero total amounts.
func (e JournalEntryPosted) HasNonZeroAmounts() (bool, map[string]any) {
	total := decimal.Zero
	for _, line := range e.Lines {
		d, c := line.BaseAmount()
		total = total.Add(d).Add(c)
	}
	return !total.IsZero(), nil
}
