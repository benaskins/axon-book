package gl

import (
	"context"
	"encoding/json"
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
	accountTypeFn func(string) AccountType // override for testing
}

// NewHandler creates an API handler for the general ledger.
func NewHandler(ledger *Ledger, accounts *ChartOfAccounts, projection *BalanceProjection) *Handler {
	h := &Handler{
		ledger:     ledger,
		accounts:   accounts,
		projection: projection,
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

type accountResponse struct {
	Number    string      `json:"number"`
	Name      string      `json:"name"`
	Type      AccountType `json:"type"`
	Parent    string      `json:"parent,omitempty"`
	Active    bool        `json:"active"`
	CreatedAt time.Time   `json:"created_at"`
}

type postEntryRequest struct {
	Date          string             `json:"date"`
	Description   string             `json:"description"`
	Lines         []postEntryLine    `json:"lines"`
	SourceType    string             `json:"source_type,omitempty"`
	SourceRef     string             `json:"source_ref,omitempty"`
	Kind          EntryKind          `json:"kind,omitempty"`
	ReversesEntry string             `json:"reverses_entry,omitempty"`
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
	var req createAccountRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	if req.Number == "" || req.Name == "" || req.Type == "" {
		http.Error(w, "number, name, and type are required", http.StatusBadRequest)
		return
	}

	switch req.Type {
	case Asset, Liability, Equity, Revenue, Expense:
	default:
		http.Error(w, "type must be asset, liability, equity, revenue, or expense", http.StatusBadRequest)
		return
	}

	acct, err := h.accounts.Create(r.Context(), req.Number, req.Name, req.Type, req.Parent)
	if err != nil {
		http.Error(w, "failed to create account", http.StatusInternalServerError)
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
		http.Error(w, "failed to list accounts", http.StatusInternalServerError)
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
		http.Error(w, "failed to get account", http.StatusInternalServerError)
		return
	}
	if acct == nil {
		http.Error(w, "not found", http.StatusNotFound)
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
		http.Error(w, "failed to deactivate account", http.StatusInternalServerError)
		return
	}
	axon.WriteJSON(w, http.StatusOK, map[string]string{"status": "deactivated", "account": number})
}

// PostEntry handles POST /api/entries
func (h *Handler) PostEntry(w http.ResponseWriter, r *http.Request) {
	var req postEntryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	date, err := time.Parse("2006-01-02", req.Date)
	if err != nil {
		http.Error(w, "date must be YYYY-MM-DD", http.StatusBadRequest)
		return
	}

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
		http.Error(w, err.Error(), http.StatusUnprocessableEntity)
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
		http.Error(w, "from and to query parameters required (YYYY-MM-DD)", http.StatusBadRequest)
		return
	}

	from, err := time.Parse("2006-01-02", fromStr)
	if err != nil {
		http.Error(w, "from must be YYYY-MM-DD", http.StatusBadRequest)
		return
	}
	to, err := time.Parse("2006-01-02", toStr)
	if err != nil {
		http.Error(w, "to must be YYYY-MM-DD", http.StatusBadRequest)
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

// RegisterRoutes registers all ledger API routes on the given mux.
// All routes are wrapped with the provided auth middleware.
func (h *Handler) RegisterRoutes(mux *http.ServeMux, requireAuth func(http.Handler) http.Handler) {
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
}
