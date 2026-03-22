package gl

import "github.com/shopspring/decimal"

// Domain event type constants for the operations stream.
const (
	EventInvestmentMade       = "investment.made"
	EventPermitPaid           = "permit.paid"
	EventSupplyPurchased      = "supply.purchased"
	EventIcePurchased         = "ice.purchased"
	EventSaleCompleted        = "sale.completed"
	EventSpoilageRecorded     = "spoilage.recorded"
	EventAdvertisingPurchased = "advertising.purchased"
)

// Stream names.
const (
	StreamOperations = "operations"
	StreamLedger     = "ledger"
)

// InvestmentMade records a capital contribution.
type InvestmentMade struct {
	Date        string          `json:"date"`
	Amount      decimal.Decimal `json:"amount"`
	Description string          `json:"description"`
}

// PermitPaid records a periodic permit fee.
type PermitPaid struct {
	Date        string          `json:"date"`
	Amount      decimal.Decimal `json:"amount"`
	Description string          `json:"description"`
}

// SupplyItem is a single item in a supply purchase.
type SupplyItem struct {
	Name     string          `json:"name"`
	Account  string          `json:"account"`
	Quantity int             `json:"quantity"`
	Cost     decimal.Decimal `json:"cost"`
}

// SupplyPurchased records a batch inventory purchase.
type SupplyPurchased struct {
	Date  string       `json:"date"`
	Items []SupplyItem `json:"items"`
	Ref   string       `json:"ref,omitempty"`
}

// IcePurchased records a daily ice purchase (perishable).
type IcePurchased struct {
	Date string          `json:"date"`
	Cost decimal.Decimal `json:"cost"`
}

// SaleCompleted records a day's lemonade sales.
type SaleCompleted struct {
	Date        string          `json:"date"`
	Cups        int             `json:"cups"`
	PricePerCup decimal.Decimal `json:"price_per_cup"`
	Weather     string          `json:"weather"`
}

// SpoilageRecorded records inventory loss.
type SpoilageRecorded struct {
	Date    string          `json:"date"`
	Item    string          `json:"item"`
	Account string          `json:"account"`
	Amount  decimal.Decimal `json:"amount"`
	Reason  string          `json:"reason"`
}

// AdvertisingPurchased records a marketing spend.
type AdvertisingPurchased struct {
	Date        string          `json:"date"`
	Amount      decimal.Decimal `json:"amount"`
	Description string          `json:"description"`
}
