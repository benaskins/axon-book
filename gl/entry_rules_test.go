package gl

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/shopspring/decimal"
)

func TestJournalEntryRules_Valid(t *testing.T) {
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

func TestJournalEntryRules_TooFewLines(t *testing.T) {
	entry := JournalEntryPosted{
		Lines: []Line{
			{Account: "1000", Debit: decimal.NewFromInt(100)},
		},
	}

	result := JournalEntryIsValid.Evaluate(entry)
	assertHasViolation(t, result, "TooFewLines")
}

func TestJournalEntryRules_Unbalanced(t *testing.T) {
	entry := JournalEntryPosted{
		Lines: []Line{
			{Account: "1000", Debit: decimal.NewFromInt(1000)},
			{Account: "4000", Credit: decimal.NewFromInt(500)},
		},
	}

	result := JournalEntryIsValid.Evaluate(entry)
	assertHasViolation(t, result, "BalanceMismatch")

	// Context should carry the totals as a typed struct
	for _, v := range result.Items {
		if v.Code == "BalanceMismatch" {
			m, ok := v.Context.(BalanceMismatch)
			if !ok {
				t.Fatalf("expected BalanceMismatch context, got %T", v.Context)
			}
			if m.TotalDebits != "1000" {
				t.Errorf("expected TotalDebits=1000, got %v", m.TotalDebits)
			}
			if m.TotalCredits != "500" {
				t.Errorf("expected TotalCredits=500, got %v", m.TotalCredits)
			}
		}
	}
}

func TestJournalEntryRules_ZeroAmounts(t *testing.T) {
	entry := JournalEntryPosted{
		Lines: []Line{
			{Account: "1000", Debit: decimal.Zero},
			{Account: "2000", Credit: decimal.Zero},
		},
	}

	result := JournalEntryIsValid.Evaluate(entry)
	assertHasViolation(t, result, "ZeroAmounts")
}

func TestJournalEntryRules_MultipleViolations(t *testing.T) {
	entry := JournalEntryPosted{
		Lines: []Line{
			{Account: "1000", Debit: decimal.Zero},
		},
	}

	result := JournalEntryIsValid.Evaluate(entry)
	assertHasViolation(t, result, "TooFewLines")
	assertHasViolation(t, result, "ZeroAmounts")
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

	assertHasViolation(t, ve.Violations, "BalanceMismatch")
}

func TestJournalEntryRules_MissingDescription(t *testing.T) {
	entry := JournalEntryPosted{
		Date: time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC),
		Lines: []Line{
			{Account: "1000", Debit: decimal.NewFromInt(100)},
			{Account: "4000", Credit: decimal.NewFromInt(100)},
		},
	}

	result := JournalEntryIsValid.Evaluate(entry)
	assertHasViolation(t, result, "MissingDescription")
}

func TestJournalEntryRules_MissingDate(t *testing.T) {
	entry := JournalEntryPosted{
		Description: "Service revenue",
		Lines: []Line{
			{Account: "1000", Debit: decimal.NewFromInt(100)},
			{Account: "4000", Credit: decimal.NewFromInt(100)},
		},
	}

	result := JournalEntryIsValid.Evaluate(entry)
	assertHasViolation(t, result, "MissingDate")
}

func assertHasViolation(t *testing.T, violations interface{ Codes() []string }, code string) {
	t.Helper()
	for _, c := range violations.Codes() {
		if c == code {
			return
		}
	}
	t.Errorf("expected violation %q, got %v", code, violations.Codes())
}
