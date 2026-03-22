// Seed populates the general ledger with a lemonade stand chart of accounts
// and a year of simulated journal entries.
//
// Based on the classic Lemonade Stand video game: buy supplies, make lemonade,
// sell cups, deal with weather. Generates realistic seasonal patterns —
// busy summers, quiet winters, weather-driven demand, spoilage, and advertising.
//
// Usage:
//
//	DATABASE_URL=postgres://... go run ./cmd/seed/
//	DATABASE_URL=postgres://... go run ./cmd/seed/ --year 2025
package main

import (
	"context"
	"fmt"
	"log/slog"
	"math/rand"
	"os"
	"time"

	"github.com/benaskins/axon"
	fact "github.com/benaskins/axon-fact"
	"github.com/benaskins/axon-book/gl"
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
	events := fact.NewPostgresStore(db, fact.WithPgProjector(projection))
	accounts := gl.NewChartOfAccounts(db)
	ledger := gl.NewLedger(events, accounts, "AUD")

	// --- Seed chart of accounts ---
	slog.Info("seeding chart of accounts")
	if err := seedAccounts(ctx, accounts); err != nil {
		return fmt.Errorf("seed accounts: %w", err)
	}

	// --- Generate journal entries ---
	slog.Info("generating journal entries", "year", year)
	stats, err := generateEntries(ctx, ledger, year)
	if err != nil {
		return fmt.Errorf("generate entries: %w", err)
	}

	slog.Info("seed complete",
		"entries", stats.entries,
		"revenue", stats.revenue.StringFixed(2),
		"expenses", stats.expenses.StringFixed(2),
	)

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
	entries  int
	revenue  decimal.Decimal
	expenses decimal.Decimal
}

// --- Chart of Accounts ---
//
// Modelled on a lemonade stand business:
//
// Assets:     Cash, Inventory (lemons, sugar, cups, ice)
// Equity:     Owner's equity, retained earnings
// Revenue:    Lemonade sales
// Expenses:   COGS, advertising, stand permit, spoilage

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

// --- Entry Generation ---
//
// Simulates a year of lemonade stand operations:
//
// - Jan 1: Owner invests starting capital
// - Monthly: Stand permit fee
// - Weekly: Buy inventory (lemons, sugar, cups)
// - Daily: Buy ice (doesn't keep), sell lemonade
// - Weekly: Advertising (signs)
// - Occasional: Spoilage from bad weather or overstock
//
// Sales volume depends on:
// - Season (summer peak, winter trough)
// - Day of week (weekends busier)
// - Weather (random hot/mild/cold/rainy days)

func generateEntries(ctx context.Context, ledger *gl.Ledger, year int) (seedStats, error) {
	rng := rand.New(rand.NewSource(int64(year)))
	var stats seedStats
	d := decimal.NewFromInt

	// --- Owner investment ---
	if _, err := ledger.Post(ctx, gl.JournalEntryPosted{
		Date:        date(year, 1, 1),
		Description: "Owner's initial investment",
		Kind:        gl.Operating,
		Lines: []gl.Line{
			{Account: "1000", Debit: d(5000)},
			{Account: "3000", Credit: d(5000)},
		},
	}); err != nil {
		return stats, fmt.Errorf("owner investment: %w", err)
	}
	stats.entries++

	start := date(year, 1, 1)
	end := date(year, 12, 31)

	for day := start; !day.After(end); day = day.AddDate(0, 0, 1) {
		month := day.Month()
		weekday := day.Weekday()

		// --- Monthly: Stand permit (1st of each month, Mar-Oct) ---
		if day.Day() == 1 && month >= time.March && month <= time.October {
			permit := d(50)
			if _, err := ledger.Post(ctx, gl.JournalEntryPosted{
				Date:        day,
				Description: fmt.Sprintf("Stand permit - %s", month),
				Kind:        gl.Operating,
				Lines: []gl.Line{
					{Account: "5500", Debit: permit},
					{Account: "1000", Credit: permit},
				},
			}); err != nil {
				return stats, err
			}
			stats.entries++
			stats.expenses = stats.expenses.Add(permit)
		}

		// --- Weekly inventory purchase (Monday) ---
		if weekday == time.Monday && month >= time.March && month <= time.October {
			seasonMultiplier := seasonFactor(month)

			lemons := d(int64(30 * seasonMultiplier))
			sugar := d(int64(15 * seasonMultiplier))
			cups := d(int64(10 * seasonMultiplier))

			total := lemons.Add(sugar).Add(cups)

			if _, err := ledger.Post(ctx, gl.JournalEntryPosted{
				Date:        day,
				Description: "Weekly supply run",
				Kind:        gl.Operating,
				SourceType:  "purchase_order",
				SourceRef:   fmt.Sprintf("po-%s", day.Format("20060102")),
				Lines: []gl.Line{
					{Account: "1100", Debit: lemons, Description: "Lemons"},
					{Account: "1200", Debit: sugar, Description: "Sugar"},
					{Account: "1300", Debit: cups, Description: "Cups"},
					{Account: "1000", Credit: total},
				},
			}); err != nil {
				return stats, err
			}
			stats.entries++
		}

		// --- Weekly advertising (Wednesday, peak season) ---
		if weekday == time.Wednesday && month >= time.April && month <= time.September {
			signs := d(int64(15 + rng.Intn(20)))
			if _, err := ledger.Post(ctx, gl.JournalEntryPosted{
				Date:        day,
				Description: "Advertising - signs and flyers",
				Kind:        gl.Operating,
				Lines: []gl.Line{
					{Account: "5400", Debit: signs},
					{Account: "1000", Credit: signs},
				},
			}); err != nil {
				return stats, err
			}
			stats.entries++
			stats.expenses = stats.expenses.Add(signs)
		}

		// --- Daily operations (only open Mar-Oct, not rainy) ---
		if month < time.March || month > time.October {
			continue
		}

		weather := randomWeather(rng, month)
		if weather == "rainy" {
			// Rainy day — spoilage of ice, no sales
			if rng.Float64() < 0.5 {
				spoilage := d(int64(5 + rng.Intn(15)))
				if _, err := ledger.Post(ctx, gl.JournalEntryPosted{
					Date:        day,
					Description: "Ice melted - rainy day, stand closed",
					Kind:        gl.Operating,
					Lines: []gl.Line{
						{Account: "5600", Debit: spoilage},
						{Account: "1400", Credit: spoilage},
					},
				}); err != nil {
					return stats, err
				}
				stats.entries++
				stats.expenses = stats.expenses.Add(spoilage)
			}
			continue
		}

		// --- Buy ice (daily, doesn't keep) ---
		iceCost := d(int64(8 + rng.Intn(7)))
		if weather == "hot" {
			iceCost = iceCost.Add(d(int64(5 + rng.Intn(5))))
		}

		if _, err := ledger.Post(ctx, gl.JournalEntryPosted{
			Date:        day,
			Description: "Daily ice purchase",
			Kind:        gl.Operating,
			Lines: []gl.Line{
				{Account: "1400", Debit: iceCost},
				{Account: "1000", Credit: iceCost},
			},
		}); err != nil {
			return stats, err
		}
		stats.entries++

		// --- Sales ---
		cups := dailyCups(rng, month, weekday, weather)
		pricePerCup := cupPrice(weather)
		salesRevenue := pricePerCup.Mul(decimal.NewFromInt(int64(cups)))

		if cups > 0 {
			if _, err := ledger.Post(ctx, gl.JournalEntryPosted{
				Date:        day,
				Description: fmt.Sprintf("Lemonade sales - %d cups @ $%s (%s)", cups, pricePerCup.StringFixed(2), weather),
				Kind:        gl.Operating,
				Lines: []gl.Line{
					{Account: "1000", Debit: salesRevenue},
					{Account: "4000", Credit: salesRevenue},
				},
			}); err != nil {
				return stats, err
			}
			stats.entries++
			stats.revenue = stats.revenue.Add(salesRevenue)

			// COGS — consume inventory
			cogsLemons := d(int64(cups)).Mul(decimal.NewFromFloat(0.50))
			cogsSugar := d(int64(cups)).Mul(decimal.NewFromFloat(0.20))
			cogsCups := d(int64(cups)).Mul(decimal.NewFromFloat(0.10))
			cogsIce := d(int64(cups)).Mul(decimal.NewFromFloat(0.30))
			totalCogs := cogsLemons.Add(cogsSugar).Add(cogsCups).Add(cogsIce)

			if _, err := ledger.Post(ctx, gl.JournalEntryPosted{
				Date:        day,
				Description: fmt.Sprintf("COGS - %d cups", cups),
				Kind:        gl.Operating,
				Lines: []gl.Line{
					{Account: "5000", Debit: cogsLemons, Description: "Lemons"},
					{Account: "5100", Debit: cogsSugar, Description: "Sugar"},
					{Account: "5200", Debit: cogsCups, Description: "Cups"},
					{Account: "5300", Debit: cogsIce, Description: "Ice"},
					{Account: "1100", Credit: cogsLemons},
					{Account: "1200", Credit: cogsSugar},
					{Account: "1300", Credit: cogsCups},
					{Account: "1400", Credit: cogsIce},
				},
			}); err != nil {
				return stats, err
			}
			stats.entries++
			stats.expenses = stats.expenses.Add(totalCogs)
		}

		// --- Occasional spoilage (overripe lemons, end of week) ---
		if weekday == time.Friday && rng.Float64() < 0.15 {
			spoilage := d(int64(5 + rng.Intn(20)))
			if _, err := ledger.Post(ctx, gl.JournalEntryPosted{
				Date:        day,
				Description: "Spoilage - overripe lemons discarded",
				Kind:        gl.Operating,
				Lines: []gl.Line{
					{Account: "5600", Debit: spoilage},
					{Account: "1100", Credit: spoilage},
				},
			}); err != nil {
				return stats, err
			}
			stats.entries++
			stats.expenses = stats.expenses.Add(spoilage)
		}
	}

	return stats, nil
}

func date(year int, month time.Month, day int) time.Time {
	return time.Date(year, month, day, 0, 0, 0, 0, time.UTC)
}

// seasonFactor returns a multiplier for inventory/sales based on month.
func seasonFactor(m time.Month) float64 {
	switch m {
	case time.June, time.July, time.August:
		return 2.0 // summer peak
	case time.May, time.September:
		return 1.5 // shoulder season
	case time.April, time.October:
		return 1.0 // early/late season
	default:
		return 0.5 // off season
	}
}

// randomWeather returns a weather condition weighted by season.
func randomWeather(rng *rand.Rand, m time.Month) string {
	r := rng.Float64()
	switch {
	case m >= time.June && m <= time.August:
		// Summer: mostly hot
		if r < 0.5 {
			return "hot"
		} else if r < 0.85 {
			return "mild"
		} else if r < 0.95 {
			return "cold"
		}
		return "rainy"
	case m >= time.April && m <= time.May || m >= time.September && m <= time.October:
		// Shoulder: mixed
		if r < 0.2 {
			return "hot"
		} else if r < 0.55 {
			return "mild"
		} else if r < 0.8 {
			return "cold"
		}
		return "rainy"
	default:
		// Off-season
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

	// Weather
	switch weather {
	case "hot":
		base *= 1.8
	case "mild":
		base *= 1.0
	case "cold":
		base *= 0.4
	}

	// Weekends busier
	if weekday == time.Saturday || weekday == time.Sunday {
		base *= 1.5
	}

	// Add some randomness (±30%)
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

