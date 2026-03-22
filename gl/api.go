package gl

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/benaskins/axon"
	"github.com/shopspring/decimal"
)

// Handler serves the general ledger REST API.
type Handler struct {
	ledger        *Ledger
	accounts      *ChartOfAccounts
	projection    *BalanceProjection
	summaries     *DailySummaryStore
	accountTypeFn func(string) AccountType // override for testing
}

// NewHandler creates an API handler for the general ledger.
func NewHandler(ledger *Ledger, accounts *ChartOfAccounts, projection *BalanceProjection, summaries *DailySummaryStore) *Handler {
	h := &Handler{
		ledger:     ledger,
		accounts:   accounts,
		projection: projection,
		summaries:  summaries,
	}
	h.accountTypeFn = h.accountTypeLookup
	return h
}

// --- Request/Response types ---

type createAccountRequest struct {
	Number string      `json:"number"`
	Name   string      `json:"name"`
	Type   AccountType `json:"type"`
	Parent string      `json:"parent,omitempty"`
}

func (r createAccountRequest) Validate() error {
	if r.Number == "" || r.Name == "" || r.Type == "" {
		return fmt.Errorf("number, name, and type are required")
	}
	switch r.Type {
	case Asset, Liability, Equity, Revenue, Expense:
	default:
		return fmt.Errorf("type must be asset, liability, equity, revenue, or expense")
	}
	return nil
}

type accountResponse struct {
	Number    string      `json:"number"`
	Name      string      `json:"name"`
	Type      AccountType `json:"type"`
	Parent    string      `json:"parent,omitempty"`
	Active    bool        `json:"active"`
	CreatedAt time.Time   `json:"created_at"`
}

type postEntryRequest struct {
	Date          string          `json:"date"`
	Description   string          `json:"description"`
	Lines         []postEntryLine `json:"lines"`
	SourceType    string          `json:"source_type,omitempty"`
	SourceRef     string          `json:"source_ref,omitempty"`
	Kind          EntryKind       `json:"kind,omitempty"`
	ReversesEntry string          `json:"reverses_entry,omitempty"`
}

func (r postEntryRequest) Validate() error {
	if r.Date == "" {
		return fmt.Errorf("date is required")
	}
	if _, err := time.Parse("2006-01-02", r.Date); err != nil {
		return fmt.Errorf("date must be YYYY-MM-DD")
	}
	if len(r.Lines) == 0 {
		return fmt.Errorf("at least one line is required")
	}
	return nil
}

type postEntryLine struct {
	Account      string `json:"account"`
	Debit        string `json:"debit,omitempty"`
	Credit       string `json:"credit,omitempty"`
	Currency     string `json:"currency,omitempty"`
	ExchangeRate string `json:"exchange_rate,omitempty"`
	Description  string `json:"description,omitempty"`
}

type postEntryResponse struct {
	EntryID string `json:"entry_id"`
}

type trialBalanceResponse struct {
	Balances     []balanceItem `json:"balances"`
	TotalDebits  string        `json:"total_debits"`
	TotalCredits string        `json:"total_credits"`
	InBalance    bool          `json:"in_balance"`
}

type balanceItem struct {
	Account string `json:"account"`
	Debits  string `json:"debits"`
	Credits string `json:"credits"`
	Net     string `json:"net"`
}

type profitAndLossResponse struct {
	From      string        `json:"from"`
	To        string        `json:"to"`
	Revenue   []balanceItem `json:"revenue"`
	Expenses  []balanceItem `json:"expenses"`
	NetIncome string        `json:"net_income"`
}

// --- Handlers ---

// CreateAccount handles POST /api/accounts
func (h *Handler) CreateAccount(w http.ResponseWriter, r *http.Request) {
	req, ok := axon.DecodeJSON[createAccountRequest](w, r)
	if !ok {
		return
	}

	acct, err := h.accounts.Create(r.Context(), req.Number, req.Name, req.Type, req.Parent)
	if err != nil {
		axon.WriteError(w, http.StatusInternalServerError, "failed to create account")
		return
	}

	axon.WriteJSON(w, http.StatusCreated, accountResponse{
		Number:    acct.Number,
		Name:      acct.Name,
		Type:      acct.Type,
		Parent:    acct.Parent,
		Active:    acct.Active,
		CreatedAt: acct.CreatedAt,
	})
}

// ListAccounts handles GET /api/accounts
func (h *Handler) ListAccounts(w http.ResponseWriter, r *http.Request) {
	accounts, err := h.accounts.List(r.Context())
	if err != nil {
		axon.WriteError(w, http.StatusInternalServerError, "failed to list accounts")
		return
	}

	resp := make([]accountResponse, len(accounts))
	for i, a := range accounts {
		resp[i] = accountResponse{
			Number:    a.Number,
			Name:      a.Name,
			Type:      a.Type,
			Parent:    a.Parent,
			Active:    a.Active,
			CreatedAt: a.CreatedAt,
		}
	}
	axon.WriteJSON(w, http.StatusOK, resp)
}

// GetAccount handles GET /api/accounts/{number}
func (h *Handler) GetAccount(w http.ResponseWriter, r *http.Request) {
	number := r.PathValue("number")
	acct, err := h.accounts.Get(r.Context(), number)
	if err != nil {
		axon.WriteError(w, http.StatusInternalServerError, "failed to get account")
		return
	}
	if acct == nil {
		axon.WriteError(w, http.StatusNotFound, "not found")
		return
	}

	axon.WriteJSON(w, http.StatusOK, accountResponse{
		Number:    acct.Number,
		Name:      acct.Name,
		Type:      acct.Type,
		Parent:    acct.Parent,
		Active:    acct.Active,
		CreatedAt: acct.CreatedAt,
	})
}

// DeactivateAccount handles DELETE /api/accounts/{number}
func (h *Handler) DeactivateAccount(w http.ResponseWriter, r *http.Request) {
	number := r.PathValue("number")
	if err := h.accounts.Deactivate(r.Context(), number); err != nil {
		axon.WriteError(w, http.StatusInternalServerError, "failed to deactivate account")
		return
	}
	axon.WriteJSON(w, http.StatusOK, map[string]string{"status": "deactivated", "account": number})
}

// PostEntry handles POST /api/entries
func (h *Handler) PostEntry(w http.ResponseWriter, r *http.Request) {
	req, ok := axon.DecodeJSON[postEntryRequest](w, r)
	if !ok {
		return
	}

	date, _ := time.Parse("2006-01-02", req.Date) // already validated

	lines := make([]Line, len(req.Lines))
	for i, l := range req.Lines {
		debit, _ := decimal.NewFromString(l.Debit)
		credit, _ := decimal.NewFromString(l.Credit)
		exchangeRate, _ := decimal.NewFromString(l.ExchangeRate)

		lines[i] = Line{
			Account:      l.Account,
			Debit:        debit,
			Credit:       credit,
			Currency:     l.Currency,
			ExchangeRate: exchangeRate,
			Description:  l.Description,
		}
	}

	entry := JournalEntryPosted{
		Date:          date,
		Description:   req.Description,
		Lines:         lines,
		SourceType:    req.SourceType,
		SourceRef:     req.SourceRef,
		Kind:          req.Kind,
		ReversesEntry: req.ReversesEntry,
	}

	entryID, err := h.ledger.Post(r.Context(), entry)
	if err != nil {
		axon.WriteError(w, http.StatusUnprocessableEntity, err.Error())
		return
	}

	axon.WriteJSON(w, http.StatusCreated, postEntryResponse{EntryID: entryID})
}

// TrialBalance handles GET /api/trial-balance
func (h *Handler) TrialBalance(w http.ResponseWriter, r *http.Request) {
	tb := h.projection.TrialBalance()

	balances := make([]balanceItem, len(tb.Balances))
	for i, b := range tb.Balances {
		balances[i] = balanceItem{
			Account: b.Account,
			Debits:  b.Debits.StringFixed(2),
			Credits: b.Credits.StringFixed(2),
			Net:     b.Net().StringFixed(2),
		}
	}

	axon.WriteJSON(w, http.StatusOK, trialBalanceResponse{
		Balances:     balances,
		TotalDebits:  tb.TotalDebits.StringFixed(2),
		TotalCredits: tb.TotalCredits.StringFixed(2),
		InBalance:    tb.InBalance(),
	})
}

// ProfitAndLoss handles GET /api/profit-and-loss?from=YYYY-MM-DD&to=YYYY-MM-DD
func (h *Handler) ProfitAndLoss(w http.ResponseWriter, r *http.Request) {
	fromStr := r.URL.Query().Get("from")
	toStr := r.URL.Query().Get("to")

	if fromStr == "" || toStr == "" {
		axon.WriteError(w, http.StatusBadRequest, "from and to query parameters required (YYYY-MM-DD)")
		return
	}

	from, err := time.Parse("2006-01-02", fromStr)
	if err != nil {
		axon.WriteError(w, http.StatusBadRequest, "from must be YYYY-MM-DD")
		return
	}
	to, err := time.Parse("2006-01-02", toStr)
	if err != nil {
		axon.WriteError(w, http.StatusBadRequest, "to must be YYYY-MM-DD")
		return
	}
	// Include the full end date
	to = to.Add(24*time.Hour - time.Nanosecond)

	pl := h.projection.ProfitAndLoss(from, to, h.accountTypeFn)

	revenue := make([]balanceItem, len(pl.Revenue))
	for i, b := range pl.Revenue {
		revenue[i] = balanceItem{
			Account: b.Account,
			Debits:  b.Debits.StringFixed(2),
			Credits: b.Credits.StringFixed(2),
			Net:     b.Credits.Sub(b.Debits).StringFixed(2),
		}
	}
	expenses := make([]balanceItem, len(pl.Expenses))
	for i, b := range pl.Expenses {
		expenses[i] = balanceItem{
			Account: b.Account,
			Debits:  b.Debits.StringFixed(2),
			Credits: b.Credits.StringFixed(2),
			Net:     b.Debits.Sub(b.Credits).StringFixed(2),
		}
	}

	axon.WriteJSON(w, http.StatusOK, profitAndLossResponse{
		From:      from.Format("2006-01-02"),
		To:        to.Format("2006-01-02"),
		Revenue:   revenue,
		Expenses:  expenses,
		NetIncome: pl.NetIncome.StringFixed(2),
	})
}

// accountTypeLookup resolves account number to type via the database.
// Used by ProfitAndLoss to classify accounts.
func (h *Handler) accountTypeLookup(number string) AccountType {
	acct, err := h.accounts.Get(context.Background(), number)
	if err != nil || acct == nil {
		return ""
	}
	return acct.Type
}

// DailySummaries handles GET /api/daily-summaries?from=YYYY-MM-DD&to=YYYY-MM-DD
func (h *Handler) DailySummaries(w http.ResponseWriter, r *http.Request) {
	from := r.URL.Query().Get("from")
	to := r.URL.Query().Get("to")

	if from == "" || to == "" {
		axon.WriteError(w, http.StatusBadRequest, "from and to query parameters required (YYYY-MM-DD)")
		return
	}

	if _, err := time.Parse("2006-01-02", from); err != nil {
		axon.WriteError(w, http.StatusBadRequest, "from must be YYYY-MM-DD")
		return
	}
	if _, err := time.Parse("2006-01-02", to); err != nil {
		axon.WriteError(w, http.StatusBadRequest, "to must be YYYY-MM-DD")
		return
	}

	summaries, err := h.summaries.List(r.Context(), from, to)
	if err != nil {
		axon.WriteError(w, http.StatusInternalServerError, "failed to query daily summaries")
		return
	}

	axon.WriteJSON(w, http.StatusOK, summaries)
}

// MonthlySummaries handles GET /api/monthly-summary
func (h *Handler) MonthlySummaries(w http.ResponseWriter, r *http.Request) {
	summaries, err := h.summaries.MonthlySummary(r.Context())
	if err != nil {
		axon.WriteError(w, http.StatusInternalServerError, "failed to query monthly summaries")
		return
	}

	axon.WriteJSON(w, http.StatusOK, summaries)
}

// RegisterRoutes registers all ledger API routes on the given mux.
// All routes are wrapped with the provided auth middleware.
// The root index endpoint is unauthenticated.
func (h *Handler) RegisterRoutes(mux *http.ServeMux, requireAuth func(http.Handler) http.Handler) {
	// Index (unauthenticated)
	mux.HandleFunc("GET /api/{$}", h.Index)

	// Chart of accounts
	mux.Handle("POST /api/accounts", requireAuth(http.HandlerFunc(h.CreateAccount)))
	mux.Handle("GET /api/accounts", requireAuth(http.HandlerFunc(h.ListAccounts)))
	mux.Handle("GET /api/accounts/{number}", requireAuth(http.HandlerFunc(h.GetAccount)))
	mux.Handle("DELETE /api/accounts/{number}", requireAuth(http.HandlerFunc(h.DeactivateAccount)))

	// Journal entries
	mux.Handle("POST /api/entries", requireAuth(http.HandlerFunc(h.PostEntry)))

	// Reports
	mux.Handle("GET /api/trial-balance", requireAuth(http.HandlerFunc(h.TrialBalance)))
	mux.Handle("GET /api/profit-and-loss", requireAuth(http.HandlerFunc(h.ProfitAndLoss)))

	// Summaries
	mux.Handle("GET /api/daily-summaries", requireAuth(http.HandlerFunc(h.DailySummaries)))
	mux.Handle("GET /api/monthly-summary", requireAuth(http.HandlerFunc(h.MonthlySummaries)))
}

// Index handles GET / — unauthenticated API index.
func (h *Handler) Index(w http.ResponseWriter, r *http.Request) {
	axon.WriteJSON(w, http.StatusOK, map[string]any{
		"service":  "axon-book",
		"currency": h.ledger.BaseCurrency(),
		"endpoints": []map[string]string{
			{"method": "GET", "path": "/health", "description": "Health check"},
			{"method": "POST", "path": "/api/accounts", "description": "Create account"},
			{"method": "GET", "path": "/api/accounts", "description": "List accounts"},
			{"method": "GET", "path": "/api/accounts/{number}", "description": "Get account"},
			{"method": "DELETE", "path": "/api/accounts/{number}", "description": "Deactivate account"},
			{"method": "POST", "path": "/api/entries", "description": "Post journal entry"},
			{"method": "GET", "path": "/api/trial-balance", "description": "Trial balance"},
			{"method": "GET", "path": "/api/profit-and-loss?from=YYYY-MM-DD&to=YYYY-MM-DD", "description": "Profit and loss"},
			{"method": "GET", "path": "/api/daily-summaries?from=YYYY-MM-DD&to=YYYY-MM-DD", "description": "Daily summaries"},
			{"method": "GET", "path": "/api/monthly-summary", "description": "Monthly summary"},
		},
	})
}
