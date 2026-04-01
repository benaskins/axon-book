package gl

import (
	"context"
	"errors"
	"testing"
	"time"

	spec "github.com/benaskins/axon-spec"
	"github.com/shopspring/decimal"
)

func TestJournalEntrySpec_Valid(t *testing.T) {
	entry := JournalEntryPosted{
		Date:        time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC),
		Description: "Service revenue",
		Lines: []Line{
			{Account: "1000", Debit: decimal.NewFromInt(1000)},
			{Account: "4000", Credit: decimal.NewFromInt(1000)},
		},
	}

	result := spec.Evaluate(entry, JournalEntryIsValid)
	if !result.IsValid() {
		t.Fatalf("expected valid, got violations: %v", result.ViolationCodes())
	}
}

func TestJournalEntrySpec_TooFewLines(t *testing.T) {
	entry := JournalEntryPosted{
		Lines: []Line{
			{Account: "1000", Debit: decimal.NewFromInt(100)},
		},
	}

	result := spec.Evaluate(entry, JournalEntryIsValid)
	assertHasViolation(t, result, MustHaveAtLeastTwoLines)
}

func TestJournalEntrySpec_Unbalanced(t *testing.T) {
	entry := JournalEntryPosted{
		Lines: []Line{
			{Account: "1000", Debit: decimal.NewFromInt(1000)},
			{Account: "4000", Credit: decimal.NewFromInt(500)},
		},
	}

	result := spec.Evaluate(entry, JournalEntryIsValid)
	assertHasViolation(t, result, DebitsMustEqualCredits)

	// Context should carry the totals
	for _, v := range result.Violations {
		if v.Code == DebitsMustEqualCredits {
			if v.Context["total_debits"] != "1000" {
				t.Errorf("expected total_debits=1000, got %v", v.Context["total_debits"])
			}
			if v.Context["total_credits"] != "500" {
				t.Errorf("expected total_credits=500, got %v", v.Context["total_credits"])
			}
		}
	}
}

func TestJournalEntrySpec_ZeroAmounts(t *testing.T) {
	entry := JournalEntryPosted{
		Lines: []Line{
			{Account: "1000", Debit: decimal.Zero},
			{Account: "2000", Credit: decimal.Zero},
		},
	}

	result := spec.Evaluate(entry, JournalEntryIsValid)
	assertHasViolation(t, result, MustHaveNonZeroAmounts)
}

func TestJournalEntrySpec_MultipleViolations(t *testing.T) {
	// Single zero-amount line: too few lines + zero amounts
	entry := JournalEntryPosted{
		Lines: []Line{
			{Account: "1000", Debit: decimal.Zero},
		},
	}

	result := spec.Evaluate(entry, JournalEntryIsValid)
	assertHasViolation(t, result, MustHaveAtLeastTwoLines)
	assertHasViolation(t, result, MustHaveNonZeroAmounts)
}

func TestViolationError_FromLedger(t *testing.T) {
	ledger, _ := newTestLedger(map[string]AccountType{
		"1000": Asset,
		"4000": Revenue,
	})

	_, err := ledger.Post(context.Background(), JournalEntryPosted{
		Date:        time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC),
		Description: "Unbalanced",
		Lines: []Line{
			{Account: "1000", Debit: decimal.NewFromInt(1000)},
			{Account: "4000", Credit: decimal.NewFromInt(500)},
		},
	})
	if err == nil {
		t.Fatal("expected error")
	}

	var ve *ViolationError
	if !errors.As(err, &ve) {
		t.Fatalf("expected ViolationError, got %T: %v", err, err)
	}

	assertHasViolation(t, ve.Result, DebitsMustEqualCredits)
}

func assertHasViolation(t *testing.T, result spec.Result, code spec.Code) {
	t.Helper()
	for _, v := range result.Violations {
		if v.Code == code {
			return
		}
	}
	t.Errorf("expected violation %q, got %v", code, result.ViolationCodes())
}
