package gl

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	fact "github.com/benaskins/axon-fact"
	"github.com/shopspring/decimal"
)

// DailySummary holds the aggregated daily operational and financial data.
type DailySummary struct {
	Date        string          `json:"date"`
	CupsSold    int             `json:"cups_sold"`
	PricePerCup decimal.Decimal `json:"price_per_cup"`
	Revenue     decimal.Decimal `json:"revenue"`
	Weather     string          `json:"weather"`
	COGSLemons  decimal.Decimal `json:"cogs_lemons"`
	COGSSugar   decimal.Decimal `json:"cogs_sugar"`
	COGSCups    decimal.Decimal `json:"cogs_cups"`
	COGSIce     decimal.Decimal `json:"cogs_ice"`
	Spoilage    decimal.Decimal `json:"spoilage"`
	Advertising decimal.Decimal `json:"advertising"`
	Permit      decimal.Decimal `json:"permit"`
	IceCost     decimal.Decimal `json:"ice_cost"`
}

// MonthlySummary aggregates daily summaries by month.
type MonthlySummary struct {
	Month       string          `json:"month"` // "2025-01"
	Revenue     decimal.Decimal `json:"revenue"`
	TotalCOGS   decimal.Decimal `json:"total_cogs"`
	Spoilage    decimal.Decimal `json:"spoilage"`
	Advertising decimal.Decimal `json:"advertising"`
	Permit      decimal.Decimal `json:"permit"`
	IceCost     decimal.Decimal `json:"ice_cost"`
	CupsSold    int             `json:"cups_sold"`
}

// DailySummaryProjection builds the daily_summaries read model from
// operations and ledger stream events. It implements fact.Projector.
type DailySummaryProjection struct {
	db *sql.DB
}

// NewDailySummaryProjection creates a projection that persists to the
// daily_summaries table via the given database connection.
func NewDailySummaryProjection(db *sql.DB) *DailySummaryProjection {
	return &DailySummaryProjection{db: db}
}

// Handle processes a single event, updating the daily_summaries table.
func (p *DailySummaryProjection) Handle(ctx context.Context, event fact.Event) error {
	switch event.Stream {
	case StreamOperations:
		return p.handleOperations(ctx, event)
	case StreamLedger:
		return p.handleLedger(ctx, event)
	default:
		return nil
	}
}

func (p *DailySummaryProjection) handleOperations(ctx context.Context, event fact.Event) error {
	switch event.Type {
	case EventSaleCompleted:
		var d SaleCompleted
		if err := json.Unmarshal(event.Data, &d); err != nil {
			return fmt.Errorf("unmarshal sale.completed: %w", err)
		}
		_, err := p.db.ExecContext(ctx, `
			INSERT INTO daily_summaries (date, cups_sold, price_per_cup, weather)
			VALUES ($1, $2, $3, $4)
			ON CONFLICT (date) DO UPDATE SET
				cups_sold = daily_summaries.cups_sold + EXCLUDED.cups_sold,
				price_per_cup = EXCLUDED.price_per_cup,
				weather = EXCLUDED.weather`,
			d.Date, d.Cups, d.PricePerCup, d.Weather)
		return err

	case EventSpoilageRecorded:
		var d SpoilageRecorded
		if err := json.Unmarshal(event.Data, &d); err != nil {
			return fmt.Errorf("unmarshal spoilage.recorded: %w", err)
		}
		_, err := p.db.ExecContext(ctx, `
			INSERT INTO daily_summaries (date, spoilage)
			VALUES ($1, $2)
			ON CONFLICT (date) DO UPDATE SET
				spoilage = daily_summaries.spoilage + EXCLUDED.spoilage`,
			d.Date, d.Amount)
		return err

	case EventAdvertisingPurchased:
		var d AdvertisingPurchased
		if err := json.Unmarshal(event.Data, &d); err != nil {
			return fmt.Errorf("unmarshal advertising.purchased: %w", err)
		}
		_, err := p.db.ExecContext(ctx, `
			INSERT INTO daily_summaries (date, advertising)
			VALUES ($1, $2)
			ON CONFLICT (date) DO UPDATE SET
				advertising = daily_summaries.advertising + EXCLUDED.advertising`,
			d.Date, d.Amount)
		return err

	case EventPermitPaid:
		var d PermitPaid
		if err := json.Unmarshal(event.Data, &d); err != nil {
			return fmt.Errorf("unmarshal permit.paid: %w", err)
		}
		_, err := p.db.ExecContext(ctx, `
			INSERT INTO daily_summaries (date, permit)
			VALUES ($1, $2)
			ON CONFLICT (date) DO UPDATE SET
				permit = daily_summaries.permit + EXCLUDED.permit`,
			d.Date, d.Amount)
		return err

	case EventIcePurchased:
		var d IcePurchased
		if err := json.Unmarshal(event.Data, &d); err != nil {
			return fmt.Errorf("unmarshal ice.purchased: %w", err)
		}
		_, err := p.db.ExecContext(ctx, `
			INSERT INTO daily_summaries (date, ice_cost)
			VALUES ($1, $2)
			ON CONFLICT (date) DO UPDATE SET
				ice_cost = daily_summaries.ice_cost + EXCLUDED.ice_cost`,
			d.Date, d.Cost)
		return err

	default:
		return nil
	}
}

func (p *DailySummaryProjection) handleLedger(ctx context.Context, event fact.Event) error {
	if event.Type != EventJournalEntryPosted {
		return nil
	}

	var entry JournalEntryPosted
	if err := json.Unmarshal(event.Data, &entry); err != nil {
		return fmt.Errorf("unmarshal journal_entry.posted: %w", err)
	}

	date := entry.Date.Format("2006-01-02")

	var revenue, cogsLemons, cogsSugar, cogsCups, cogsIce decimal.Decimal
	for _, line := range entry.Lines {
		switch line.Account {
		case "4000":
			revenue = revenue.Add(line.Credit)
		case "5000":
			cogsLemons = cogsLemons.Add(line.Debit)
		case "5100":
			cogsSugar = cogsSugar.Add(line.Debit)
		case "5200":
			cogsCups = cogsCups.Add(line.Debit)
		case "5300":
			cogsIce = cogsIce.Add(line.Debit)
		}
	}

	// Only upsert if there are relevant amounts
	if revenue.IsZero() && cogsLemons.IsZero() && cogsSugar.IsZero() && cogsCups.IsZero() && cogsIce.IsZero() {
		return nil
	}

	_, err := p.db.ExecContext(ctx, `
		INSERT INTO daily_summaries (date, revenue, cogs_lemons, cogs_sugar, cogs_cups, cogs_ice)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (date) DO UPDATE SET
			revenue = daily_summaries.revenue + EXCLUDED.revenue,
			cogs_lemons = daily_summaries.cogs_lemons + EXCLUDED.cogs_lemons,
			cogs_sugar = daily_summaries.cogs_sugar + EXCLUDED.cogs_sugar,
			cogs_cups = daily_summaries.cogs_cups + EXCLUDED.cogs_cups,
			cogs_ice = daily_summaries.cogs_ice + EXCLUDED.cogs_ice`,
		date, revenue, cogsLemons, cogsSugar, cogsCups, cogsIce)
	return err
}

// Verify DailySummaryProjection satisfies Projector at compile time.
var _ fact.Projector = (*DailySummaryProjection)(nil)

// DailySummaryStore queries the daily_summaries read model.
type DailySummaryStore struct {
	db *sql.DB
}

// NewDailySummaryStore creates a store for querying daily summaries.
func NewDailySummaryStore(db *sql.DB) *DailySummaryStore {
	return &DailySummaryStore{db: db}
}

// List returns daily summaries for the given date range (inclusive).
func (s *DailySummaryStore) List(ctx context.Context, from, to string) ([]DailySummary, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT date, cups_sold, price_per_cup, revenue, weather,
		       cogs_lemons, cogs_sugar, cogs_cups, cogs_ice,
		       spoilage, advertising, permit, ice_cost
		FROM daily_summaries
		WHERE date >= $1 AND date <= $2
		ORDER BY date`, from, to)
	if err != nil {
		return nil, fmt.Errorf("query daily summaries: %w", err)
	}
	defer rows.Close()

	var summaries []DailySummary
	for rows.Next() {
		var ds DailySummary
		if err := rows.Scan(
			&ds.Date, &ds.CupsSold, &ds.PricePerCup, &ds.Revenue, &ds.Weather,
			&ds.COGSLemons, &ds.COGSSugar, &ds.COGSCups, &ds.COGSIce,
			&ds.Spoilage, &ds.Advertising, &ds.Permit, &ds.IceCost,
		); err != nil {
			return nil, fmt.Errorf("scan daily summary: %w", err)
		}
		summaries = append(summaries, ds)
	}
	return summaries, rows.Err()
}

// MonthlySummary returns aggregated summaries grouped by month.
func (s *DailySummaryStore) MonthlySummary(ctx context.Context) ([]MonthlySummary, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT to_char(date, 'YYYY-MM') AS month,
		       SUM(revenue),
		       SUM(cogs_lemons + cogs_sugar + cogs_cups + cogs_ice),
		       SUM(spoilage),
		       SUM(advertising),
		       SUM(permit),
		       SUM(ice_cost),
		       SUM(cups_sold)
		FROM daily_summaries
		GROUP BY to_char(date, 'YYYY-MM')
		ORDER BY month`)
	if err != nil {
		return nil, fmt.Errorf("query monthly summaries: %w", err)
	}
	defer rows.Close()

	var summaries []MonthlySummary
	for rows.Next() {
		var ms MonthlySummary
		if err := rows.Scan(
			&ms.Month, &ms.Revenue, &ms.TotalCOGS,
			&ms.Spoilage, &ms.Advertising, &ms.Permit,
			&ms.IceCost, &ms.CupsSold,
		); err != nil {
			return nil, fmt.Errorf("scan monthly summary: %w", err)
		}
		summaries = append(summaries, ms)
	}
	return summaries, rows.Err()
}
