// Seed populates the general ledger with a lemonade stand chart of accounts
// and a year of simulated operations via domain events.
//
// Domain events (sale.completed, supply.purchased, etc.) are emitted to the
// operations stream. The Reactor projector automatically translates them into
// journal entries on the ledger stream — both within the same transaction.
//
// Usage:
//
//	DATABASE_URL=postgres://... go run ./cmd/seed/
//	DATABASE_URL=postgres://... go run ./cmd/seed/ --year 2025
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"math/rand"
	"os"
	"time"

	"github.com/benaskins/axon"
	fact "github.com/benaskins/axon-fact"
	"github.com/benaskins/axon-book/gl"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

func main() {
	if err := run(); err != nil {
		slog.Error("seed failed", "error", err)
		os.Exit(1)
	}
}

func run() error {
	ctx := context.Background()

	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		return fmt.Errorf("DATABASE_URL must be set")
	}

	year := 2025
	if len(os.Args) > 2 && os.Args[1] == "--year" {
		fmt.Sscanf(os.Args[2], "%d", &year)
	}

	// --- Database ---
	db, err := axon.OpenDB(dsn, "book")
	if err != nil {
		return fmt.Errorf("open database: %w", err)
	}
	if err := axon.RunMigrations(db, fact.Migrations); err != nil {
		return fmt.Errorf("run event migrations: %w", err)
	}
	if err := axon.RunMigrations(db, gl.Migrations); err != nil {
		return fmt.Errorf("run gl migrations: %w", err)
	}

	// --- Domain ---
	projection := gl.NewBalanceProjection()
	accounts := gl.NewChartOfAccounts(db)

	// Wire up the reactor: domain events → journal entries
	var store *fact.PostgresStore
	reactor := gl.NewReactor(nil) // placeholder, set ledger after store creation
	store = fact.NewPostgresStore(db,
		fact.WithPgProjector(reactor),
		fact.WithPgProjector(projection),
	)
	ledger := gl.NewLedger(store, accounts, "AUD")
	reactor.SetLedger(ledger)

	// --- Seed chart of accounts ---
	slog.Info("seeding chart of accounts")
	if err := seedAccounts(ctx, accounts); err != nil {
		return fmt.Errorf("seed accounts: %w", err)
	}

	// --- Generate domain events ---
	slog.Info("generating operations", "year", year)
	stats, err := generateEvents(ctx, store, year)
	if err != nil {
		return fmt.Errorf("generate events: %w", err)
	}

	slog.Info("seed complete", "domain_events", stats.events)

	// --- Print trial balance ---
	tb := projection.TrialBalance()
	fmt.Println("\n=== Trial Balance ===")
	for _, b := range tb.Balances {
		fmt.Printf("  %-6s  DR: %10s  CR: %10s\n", b.Account, b.Debits.StringFixed(2), b.Credits.StringFixed(2))
	}
	fmt.Printf("  %-6s  DR: %10s  CR: %10s  Balanced: %v\n",
		"TOTAL", tb.TotalDebits.StringFixed(2), tb.TotalCredits.StringFixed(2), tb.InBalance())

	return nil
}

type seedStats struct {
	events int
}

// --- Chart of Accounts ---

var chartOfAccounts = []struct {
	number string
	name   string
	typ    gl.AccountType
	parent string
}{
	// Assets
	{"1000", "Cash", gl.Asset, ""},
	{"1100", "Inventory - Lemons", gl.Asset, ""},
	{"1200", "Inventory - Sugar", gl.Asset, ""},
	{"1300", "Inventory - Cups", gl.Asset, ""},
	{"1400", "Inventory - Ice", gl.Asset, ""},

	// Equity
	{"3000", "Owner's Equity", gl.Equity, ""},
	{"3100", "Retained Earnings", gl.Equity, ""},

	// Revenue
	{"4000", "Lemonade Sales", gl.Revenue, ""},

	// Expenses
	{"5000", "COGS - Lemons", gl.Expense, ""},
	{"5100", "COGS - Sugar", gl.Expense, ""},
	{"5200", "COGS - Cups", gl.Expense, ""},
	{"5300", "COGS - Ice", gl.Expense, ""},
	{"5400", "Advertising", gl.Expense, ""},
	{"5500", "Stand Permit", gl.Expense, ""},
	{"5600", "Spoilage", gl.Expense, ""},
}

func seedAccounts(ctx context.Context, coa *gl.ChartOfAccounts) error {
	for _, a := range chartOfAccounts {
		existing, err := coa.Get(ctx, a.number)
		if err != nil {
			return err
		}
		if existing != nil {
			slog.Info("account exists, skipping", "number", a.number, "name", a.name)
			continue
		}
		if _, err := coa.Create(ctx, a.number, a.name, a.typ, a.parent); err != nil {
			return fmt.Errorf("create %s %s: %w", a.number, a.name, err)
		}
		slog.Info("created account", "number", a.number, "name", a.name)
	}
	return nil
}

// --- Event Generation ---
//
// Emits domain events to the operations stream. The Reactor projector
// handles translation into journal entries automatically.

func generateEvents(ctx context.Context, store fact.EventStore, year int) (seedStats, error) {
	rng := rand.New(rand.NewSource(int64(year)))
	var stats seedStats

	// --- Owner investment ---
	if err := emit(ctx, store, gl.EventInvestmentMade, gl.InvestmentMade{
		Date:        dateStr(year, 1, 1),
		Amount:      decimal.NewFromInt(5000),
		Description: "Owner's initial investment",
	}); err != nil {
		return stats, fmt.Errorf("owner investment: %w", err)
	}
	stats.events++

	start := date(year, 1, 1)
	end := date(year, 12, 31)

	for day := start; !day.After(end); day = day.AddDate(0, 0, 1) {
		month := day.Month()
		weekday := day.Weekday()
		ds := day.Format("2006-01-02")

		// --- Monthly: Stand permit (1st of each month, Mar-Oct) ---
		if day.Day() == 1 && month >= time.March && month <= time.October {
			if err := emit(ctx, store, gl.EventPermitPaid, gl.PermitPaid{
				Date:        ds,
				Amount:      decimal.NewFromInt(50),
				Description: fmt.Sprintf("Stand permit - %s", month),
			}); err != nil {
				return stats, err
			}
			stats.events++
		}

		// --- Weekly inventory purchase (Monday) ---
		if weekday == time.Monday && month >= time.March && month <= time.October {
			sf := seasonFactor(month)
			if err := emit(ctx, store, gl.EventSupplyPurchased, gl.SupplyPurchased{
				Date: ds,
				Ref:  fmt.Sprintf("po-%s", day.Format("20060102")),
				Items: []gl.SupplyItem{
					{Name: "Lemons", Account: "1100", Quantity: int(30 * sf), Cost: decimal.NewFromInt(int64(30 * sf))},
					{Name: "Sugar", Account: "1200", Quantity: int(15 * sf), Cost: decimal.NewFromInt(int64(15 * sf))},
					{Name: "Cups", Account: "1300", Quantity: int(100 * sf), Cost: decimal.NewFromInt(int64(10 * sf))},
				},
			}); err != nil {
				return stats, err
			}
			stats.events++
		}

		// --- Weekly advertising (Wednesday, peak season) ---
		if weekday == time.Wednesday && month >= time.April && month <= time.September {
			if err := emit(ctx, store, gl.EventAdvertisingPurchased, gl.AdvertisingPurchased{
				Date:        ds,
				Amount:      decimal.NewFromInt(int64(15 + rng.Intn(20))),
				Description: "Advertising - signs and flyers",
			}); err != nil {
				return stats, err
			}
			stats.events++
		}

		// --- Daily operations (only open Mar-Oct, not rainy) ---
		if month < time.March || month > time.October {
			continue
		}

		weather := randomWeather(rng, month)
		if weather == "rainy" {
			// Rainy day — possible spoilage of ice, no sales
			if rng.Float64() < 0.5 {
				if err := emit(ctx, store, gl.EventSpoilageRecorded, gl.SpoilageRecorded{
					Date:    ds,
					Item:    "ice",
					Account: "1400",
					Amount:  decimal.NewFromInt(int64(5 + rng.Intn(15))),
					Reason:  "rain",
				}); err != nil {
					return stats, err
				}
				stats.events++
			}
			continue
		}

		// --- Buy ice (daily, doesn't keep) ---
		iceCost := decimal.NewFromInt(int64(8 + rng.Intn(7)))
		if weather == "hot" {
			iceCost = iceCost.Add(decimal.NewFromInt(int64(5 + rng.Intn(5))))
		}

		if err := emit(ctx, store, gl.EventIcePurchased, gl.IcePurchased{
			Date: ds,
			Cost: iceCost,
		}); err != nil {
			return stats, err
		}
		stats.events++

		// --- Sales ---
		cups := dailyCups(rng, month, weekday, weather)
		if cups > 0 {
			if err := emit(ctx, store, gl.EventSaleCompleted, gl.SaleCompleted{
				Date:        ds,
				Cups:        cups,
				PricePerCup: cupPrice(weather),
				Weather:     weather,
			}); err != nil {
				return stats, err
			}
			stats.events++
		}

		// --- Occasional spoilage (overripe lemons, end of week) ---
		if weekday == time.Friday && rng.Float64() < 0.15 {
			if err := emit(ctx, store, gl.EventSpoilageRecorded, gl.SpoilageRecorded{
				Date:    ds,
				Item:    "lemons",
				Account: "1100",
				Amount:  decimal.NewFromInt(int64(5 + rng.Intn(20))),
				Reason:  "overripe",
			}); err != nil {
				return stats, err
			}
			stats.events++
		}
	}

	return stats, nil
}

// emit marshals event data and appends it to the operations stream.
func emit(ctx context.Context, store fact.EventStore, eventType string, data any) error {
	payload, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("marshal %s: %w", eventType, err)
	}
	return store.Append(ctx, gl.StreamOperations, []fact.Event{
		{ID: uuid.New().String(), Type: eventType, Data: payload},
	})
}

func date(year int, month time.Month, day int) time.Time {
	return time.Date(year, month, day, 0, 0, 0, 0, time.UTC)
}

func dateStr(year int, month time.Month, day int) string {
	return date(year, month, day).Format("2006-01-02")
}

// seasonFactor returns a multiplier for inventory/sales based on month.
func seasonFactor(m time.Month) float64 {
	switch m {
	case time.June, time.July, time.August:
		return 2.0
	case time.May, time.September:
		return 1.5
	case time.April, time.October:
		return 1.0
	default:
		return 0.5
	}
}

// randomWeather returns a weather condition weighted by season.
func randomWeather(rng *rand.Rand, m time.Month) string {
	r := rng.Float64()
	switch {
	case m >= time.June && m <= time.August:
		if r < 0.5 {
			return "hot"
		} else if r < 0.85 {
			return "mild"
		} else if r < 0.95 {
			return "cold"
		}
		return "rainy"
	case m >= time.April && m <= time.May || m >= time.September && m <= time.October:
		if r < 0.2 {
			return "hot"
		} else if r < 0.55 {
			return "mild"
		} else if r < 0.8 {
			return "cold"
		}
		return "rainy"
	default:
		if r < 0.1 {
			return "mild"
		} else if r < 0.5 {
			return "cold"
		}
		return "rainy"
	}
}

// dailyCups returns how many cups sold based on conditions.
func dailyCups(rng *rand.Rand, month time.Month, weekday time.Weekday, weather string) int {
	base := 20.0
	base *= seasonFactor(month)

	switch weather {
	case "hot":
		base *= 1.8
	case "mild":
		base *= 1.0
	case "cold":
		base *= 0.4
	}

	if weekday == time.Saturday || weekday == time.Sunday {
		base *= 1.5
	}

	jitter := 0.7 + rng.Float64()*0.6
	cups := int(base * jitter)
	if cups < 0 {
		cups = 0
	}
	return cups
}

// cupPrice returns price per cup based on weather.
func cupPrice(weather string) decimal.Decimal {
	switch weather {
	case "hot":
		return decimal.NewFromFloat(3.50)
	case "mild":
		return decimal.NewFromFloat(2.50)
	case "cold":
		return decimal.NewFromFloat(2.00)
	default:
		return decimal.NewFromFloat(2.50)
	}
}
