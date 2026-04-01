package gl

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/benaskins/axon-rule"
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

	result := JournalEntryIsValid.Evaluate(entry)
	if !result.IsValid() {
		t.Fatalf("expected valid, got violations: %v", result.Codes())
	}
}

func TestJournalEntrySpec_TooFewLines(t *testing.T) {
	entry := JournalEntryPosted{
		Lines: []Line{
			{Account: "1000", Debit: decimal.NewFromInt(100)},
		},
	}

	result := JournalEntryIsValid.Evaluate(entry)
	assertHasViolation(t, result, MustHaveAtLeastTwoLines)
}

func TestJournalEntrySpec_Unbalanced(t *testing.T) {
	entry := JournalEntryPosted{
		Lines: []Line{
			{Account: "1000", Debit: decimal.NewFromInt(1000)},
			{Account: "4000", Credit: decimal.NewFromInt(500)},
		},
	}

	result := JournalEntryIsValid.Evaluate(entry)
	assertHasViolation(t, result, DebitsMustEqualCredits)

	// Context should carry the totals
	for _, v := range result.Items {
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

	result := JournalEntryIsValid.Evaluate(entry)
	assertHasViolation(t, result, MustHaveNonZeroAmounts)
}

func TestJournalEntrySpec_MultipleViolations(t *testing.T) {
	entry := JournalEntryPosted{
		Lines: []Line{
			{Account: "1000", Debit: decimal.Zero},
		},
	}

	result := JournalEntryIsValid.Evaluate(entry)
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

	assertHasViolation(t, ve.Violations, DebitsMustEqualCredits)
}

func assertHasViolation(t *testing.T, violations rule.Violations, code rule.Code) {
	t.Helper()
	for _, v := range violations.Items {
		if v.Code == code {
			return
		}
	}
	t.Errorf("expected violation %q, got %v", code, violations.Codes())
}
