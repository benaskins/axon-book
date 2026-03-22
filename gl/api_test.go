package gl

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	fact "github.com/benaskins/axon-fact"
	"github.com/shopspring/decimal"
)

func newTestHandler(accounts map[string]AccountType) (*Handler, *http.ServeMux) {
	projection := NewBalanceProjection()
	store := fact.NewMemoryStore(fact.WithProjector(projection))
	mock := &mockAccounts{active: accounts}
	ledger := NewLedger(store, mock, "AUD")

	handler := &Handler{
		ledger:     ledger,
		accounts:   nil, // chart of accounts CRUD not tested here (needs Postgres)
		projection: projection,
		summaries:  nil, // daily summaries not tested here (needs Postgres)
		events:     store,
	}

	mux := http.NewServeMux()
	// Register without auth for testing
	noAuth := func(h http.Handler) http.Handler { return h }
	handler.RegisterRoutes(mux, noAuth)

	return handler, mux
}

func TestAPI_PostEntry(t *testing.T) {
	_, mux := newTestHandler(map[string]AccountType{
		"1000": Asset,
		"4000": Revenue,
	})

	body := `{
		"date": "2026-03-01",
		"description": "Test revenue",
		"lines": [
			{"account": "1000", "debit": "1000.00"},
			{"account": "4000", "credit": "1000.00"}
		]
	}`

	req := httptest.NewRequest("POST", "/api/entries", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("status = %d, want 201; body: %s", w.Code, w.Body.String())
	}

	var resp postEntryResponse
	json.NewDecoder(w.Body).Decode(&resp)
	if resp.EntryID == "" {
		t.Error("expected non-empty entry_id")
	}
}

func TestAPI_PostEntry_Unbalanced(t *testing.T) {
	_, mux := newTestHandler(map[string]AccountType{
		"1000": Asset,
		"4000": Revenue,
	})

	body := `{
		"date": "2026-03-01",
		"description": "Bad entry",
		"lines": [
			{"account": "1000", "debit": "1000.00"},
			{"account": "4000", "credit": "500.00"}
		]
	}`

	req := httptest.NewRequest("POST", "/api/entries", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusUnprocessableEntity {
		t.Errorf("status = %d, want 422", w.Code)
	}
}

func TestAPI_PostEntry_InvalidAccount(t *testing.T) {
	_, mux := newTestHandler(map[string]AccountType{
		"1000": Asset,
	})

	body := `{
		"date": "2026-03-01",
		"description": "Bad account",
		"lines": [
			{"account": "1000", "debit": "100.00"},
			{"account": "9999", "credit": "100.00"}
		]
	}`

	req := httptest.NewRequest("POST", "/api/entries", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusUnprocessableEntity {
		t.Errorf("status = %d, want 422", w.Code)
	}
}

func TestAPI_PostEntry_BadDate(t *testing.T) {
	_, mux := newTestHandler(map[string]AccountType{
		"1000": Asset,
		"4000": Revenue,
	})

	body := `{
		"date": "not-a-date",
		"description": "Bad date",
		"lines": [
			{"account": "1000", "debit": "100.00"},
			{"account": "4000", "credit": "100.00"}
		]
	}`

	req := httptest.NewRequest("POST", "/api/entries", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusUnprocessableEntity {
		t.Errorf("status = %d, want 422", w.Code)
	}
}

func TestAPI_TrialBalance(t *testing.T) {
	_, mux := newTestHandler(map[string]AccountType{
		"1000": Asset,
		"4000": Revenue,
		"5000": Expense,
	})

	// Post two entries
	postJSON(t, mux, "/api/entries", `{
		"date": "2026-03-01",
		"description": "Revenue",
		"lines": [
			{"account": "1000", "debit": "5000.00"},
			{"account": "4000", "credit": "5000.00"}
		]
	}`)
	postJSON(t, mux, "/api/entries", `{
		"date": "2026-03-05",
		"description": "Hosting",
		"lines": [
			{"account": "5000", "debit": "200.00"},
			{"account": "1000", "credit": "200.00"}
		]
	}`)

	req := httptest.NewRequest("GET", "/api/trial-balance", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body: %s", w.Code, w.Body.String())
	}

	var resp trialBalanceResponse
	json.NewDecoder(w.Body).Decode(&resp)

	if !resp.InBalance {
		t.Errorf("trial balance not in balance: debits=%s credits=%s", resp.TotalDebits, resp.TotalCredits)
	}
	if resp.TotalDebits != "5200.00" {
		t.Errorf("total debits = %s, want 5200.00", resp.TotalDebits)
	}
}

func TestAPI_ProfitAndLoss(t *testing.T) {
	accounts := map[string]AccountType{
		"1000": Asset,
		"4000": Revenue,
		"5000": Expense,
	}
	handler, mux := newTestHandler(accounts)

	// Wire up the account type lookup for P&L
	handler.accounts = nil // not used directly; lookup comes from mock
	mock := &mockAccounts{active: accounts}
	handler.accountTypeFn = mock.accountType

	postJSON(t, mux, "/api/entries", `{
		"date": "2026-03-01",
		"description": "Revenue",
		"lines": [
			{"account": "1000", "debit": "3000.00"},
			{"account": "4000", "credit": "3000.00"}
		]
	}`)
	postJSON(t, mux, "/api/entries", `{
		"date": "2026-03-10",
		"description": "Hosting",
		"lines": [
			{"account": "5000", "debit": "500.00"},
			{"account": "1000", "credit": "500.00"}
		]
	}`)

	req := httptest.NewRequest("GET", "/api/profit-and-loss?from=2026-03-01&to=2026-03-31", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body: %s", w.Code, w.Body.String())
	}

	var resp profitAndLossResponse
	json.NewDecoder(w.Body).Decode(&resp)

	if resp.NetIncome != "2500.00" {
		t.Errorf("net income = %s, want 2500.00", resp.NetIncome)
	}
}

func TestAPI_PostEntry_MultiCurrency(t *testing.T) {
	_, mux := newTestHandler(map[string]AccountType{
		"1000": Asset,
		"5100": Expense,
	})

	body := `{
		"date": "2026-03-15",
		"description": "USD subscription",
		"lines": [
			{"account": "5100", "debit": "10.00", "currency": "USD", "exchange_rate": "1.55"},
			{"account": "1000", "credit": "10.00", "currency": "USD", "exchange_rate": "1.55"}
		]
	}`

	req := httptest.NewRequest("POST", "/api/entries", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("status = %d, want 201; body: %s", w.Code, w.Body.String())
	}
}

func TestAPI_ProfitAndLoss_MissingParams(t *testing.T) {
	_, mux := newTestHandler(map[string]AccountType{})

	req := httptest.NewRequest("GET", "/api/profit-and-loss", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", w.Code)
	}
}

// postJSON is a test helper that posts JSON and asserts 201.
func postJSON(t *testing.T, mux *http.ServeMux, path, body string) {
	t.Helper()
	req := httptest.NewRequest("POST", path, bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("POST %s: status = %d, want 201; body: %s", path, w.Code, w.Body.String())
	}
}

func TestAPI_Events(t *testing.T) {
	// Manually populate both streams to avoid MemoryStore mutex deadlock
	// (reactor calls Append inside Append's projector loop).
	store := fact.NewMemoryStore()
	handler := &Handler{events: store}
	mux := http.NewServeMux()
	noAuth := func(h http.Handler) http.Handler { return h }
	handler.RegisterRoutes(mux, noAuth)

	ctx := context.Background()
	opID1 := "op-001"
	opID2 := "op-002"

	// Operations stream
	store.Append(ctx, StreamOperations, []fact.Event{
		{ID: opID1, Type: EventInvestmentMade, Data: json.RawMessage(`{"date":"2025-01-01","amount":"5000","description":"Initial investment"}`)},
		{ID: opID2, Type: EventSaleCompleted, Data: json.RawMessage(`{"date":"2025-03-01","cups":10,"price_per_cup":"3.50","weather":"warm"}`)},
	})

	// Ledger stream with causation links
	entryData := json.RawMessage(`{"entry_id":"je-001","date":"2025-01-01T00:00:00Z","description":"Initial investment","lines":[{"account":"1000","debit":"5000"},{"account":"3000","credit":"5000"}]}`)
	store.Append(ctx, StreamLedger, []fact.Event{
		{ID: "je-001", Type: "journal_entry.posted", Data: entryData, Metadata: map[string]string{"causation_id": opID1}},
	})

	req := httptest.NewRequest("GET", "/api/events", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body: %s", w.Code, w.Body.String())
	}

	var events []struct {
		ID           string           `json:"id"`
		Type         string           `json:"type"`
		Data         json.RawMessage  `json:"data"`
		JournalEntry *json.RawMessage `json:"journal_entry"`
	}
	if err := json.NewDecoder(w.Body).Decode(&events); err != nil {
		t.Fatalf("decode: %v", err)
	}

	if len(events) != 2 {
		t.Fatalf("got %d events, want 2", len(events))
	}

	if events[0].Type != EventInvestmentMade {
		t.Errorf("events[0].type = %s, want %s", events[0].Type, EventInvestmentMade)
	}
	if events[0].JournalEntry == nil {
		t.Error("events[0] should have a linked journal entry")
	}

	if events[1].Type != EventSaleCompleted {
		t.Errorf("events[1].type = %s, want %s", events[1].Type, EventSaleCompleted)
	}
	if events[1].JournalEntry != nil {
		t.Error("events[1] should NOT have a journal entry (none linked)")
	}

	// Verify journal entry content
	var entry JournalEntryPosted
	if err := json.Unmarshal(*events[0].JournalEntry, &entry); err != nil {
		t.Fatalf("unmarshal journal entry: %v", err)
	}
	if len(entry.Lines) != 2 {
		t.Errorf("investment entry has %d lines, want 2", len(entry.Lines))
	}
}

func TestAPI_Events_WithLimit(t *testing.T) {
	store := fact.NewMemoryStore()
	handler := &Handler{events: store}
	mux := http.NewServeMux()
	noAuth := func(h http.Handler) http.Handler { return h }
	handler.RegisterRoutes(mux, noAuth)

	ctx := context.Background()
	var batch []fact.Event
	for i := 0; i < 5; i++ {
		batch = append(batch, fact.Event{
			Type: EventInvestmentMade,
			Data: json.RawMessage(`{"date":"2025-01-01","amount":"1000","description":"Investment"}`),
		})
	}
	store.Append(ctx, StreamOperations, batch)

	req := httptest.NewRequest("GET", "/api/events?limit=3", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body: %s", w.Code, w.Body.String())
	}

	var events []json.RawMessage
	json.NewDecoder(w.Body).Decode(&events)
	if len(events) != 3 {
		t.Errorf("got %d events, want 3", len(events))
	}
}

// Suppress unused import warning for decimal in this file.
var _ = decimal.Zero
