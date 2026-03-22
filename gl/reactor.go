package gl

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	fact "github.com/benaskins/axon-fact"
	"github.com/shopspring/decimal"
)

// COGS rates per cup (accounting policy).
var (
	cogsLemonsPerCup = decimal.NewFromFloat(0.50)
	cogsSugarPerCup  = decimal.NewFromFloat(0.20)
	cogsCupsPerCup   = decimal.NewFromFloat(0.10)
	cogsIcePerCup    = decimal.NewFromFloat(0.30)
)

// Reactor translates domain events on the operations stream into
// journal entries on the ledger stream. It implements fact.Projector
// and uses the nested Append capability to post entries atomically
// within the same transaction as the originating domain event.
type Reactor struct {
	ledger *Ledger
}

// NewReactor creates a reactor that posts journal entries via the given ledger.
func NewReactor(ledger *Ledger) *Reactor {
	return &Reactor{ledger: ledger}
}

// Handle processes a single event. Only operations stream events are acted on;
// all others (including ledger events) are ignored.
func (r *Reactor) Handle(ctx context.Context, e fact.Event) error {
	if e.Stream != StreamOperations {
		return nil
	}

	switch e.Type {
	case EventInvestmentMade:
		return handle[InvestmentMade](e, func(d InvestmentMade) (JournalEntryPosted, error) {
			return JournalEntryPosted{
				Date:        mustParseDate(d.Date),
				Description: d.Description,
				Kind:        Operating,
				Lines: []Line{
					{Account: "1000", Debit: d.Amount},
					{Account: "3000", Credit: d.Amount},
				},
			}, nil
		}, ctx, r.ledger, e.ID)

	case EventPermitPaid:
		return handle[PermitPaid](e, func(d PermitPaid) (JournalEntryPosted, error) {
			return JournalEntryPosted{
				Date:        mustParseDate(d.Date),
				Description: d.Description,
				Kind:        Operating,
				Lines: []Line{
					{Account: "5500", Debit: d.Amount},
					{Account: "1000", Credit: d.Amount},
				},
			}, nil
		}, ctx, r.ledger, e.ID)

	case EventSupplyPurchased:
		return handle[SupplyPurchased](e, func(d SupplyPurchased) (JournalEntryPosted, error) {
			var lines []Line
			total := decimal.Zero
			for _, item := range d.Items {
				lines = append(lines, Line{
					Account:     item.Account,
					Debit:       item.Cost,
					Description: item.Name,
				})
				total = total.Add(item.Cost)
			}
			lines = append(lines, Line{Account: "1000", Credit: total})
			return JournalEntryPosted{
				Date:        mustParseDate(d.Date),
				Description: "Weekly supply run",
				Kind:        Operating,
				SourceType:  "purchase_order",
				SourceRef:   d.Ref,
				Lines:       lines,
			}, nil
		}, ctx, r.ledger, e.ID)

	case EventIcePurchased:
		return handle[IcePurchased](e, func(d IcePurchased) (JournalEntryPosted, error) {
			return JournalEntryPosted{
				Date:        mustParseDate(d.Date),
				Description: "Daily ice purchase",
				Kind:        Operating,
				Lines: []Line{
					{Account: "1400", Debit: d.Cost},
					{Account: "1000", Credit: d.Cost},
				},
			}, nil
		}, ctx, r.ledger, e.ID)

	case EventSaleCompleted:
		return handle[SaleCompleted](e, func(d SaleCompleted) (JournalEntryPosted, error) {
			cups := decimal.NewFromInt(int64(d.Cups))
			revenue := d.PricePerCup.Mul(cups)

			cogsLemons := cogsLemonsPerCup.Mul(cups)
			cogsSugar := cogsSugarPerCup.Mul(cups)
			cogsCups := cogsCupsPerCup.Mul(cups)
			cogsIce := cogsIcePerCup.Mul(cups)

			return JournalEntryPosted{
				Date:        mustParseDate(d.Date),
				Description: fmt.Sprintf("Lemonade sales - %d cups @ $%s (%s)", d.Cups, d.PricePerCup.StringFixed(2), d.Weather),
				Kind:        Operating,
				Lines: []Line{
					// Revenue
					{Account: "1000", Debit: revenue},
					{Account: "4000", Credit: revenue},
					// COGS
					{Account: "5000", Debit: cogsLemons, Description: "Lemons"},
					{Account: "5100", Debit: cogsSugar, Description: "Sugar"},
					{Account: "5200", Debit: cogsCups, Description: "Cups"},
					{Account: "5300", Debit: cogsIce, Description: "Ice"},
					{Account: "1100", Credit: cogsLemons},
					{Account: "1200", Credit: cogsSugar},
					{Account: "1300", Credit: cogsCups},
					{Account: "1400", Credit: cogsIce},
				},
			}, nil
		}, ctx, r.ledger, e.ID)

	case EventSpoilageRecorded:
		return handle[SpoilageRecorded](e, func(d SpoilageRecorded) (JournalEntryPosted, error) {
			return JournalEntryPosted{
				Date:        mustParseDate(d.Date),
				Description: fmt.Sprintf("Spoilage - %s (%s)", d.Item, d.Reason),
				Kind:        Operating,
				Lines: []Line{
					{Account: "5600", Debit: d.Amount},
					{Account: d.Account, Credit: d.Amount},
				},
			}, nil
		}, ctx, r.ledger, e.ID)

	case EventAdvertisingPurchased:
		return handle[AdvertisingPurchased](e, func(d AdvertisingPurchased) (JournalEntryPosted, error) {
			return JournalEntryPosted{
				Date:        mustParseDate(d.Date),
				Description: d.Description,
				Kind:        Operating,
				Lines: []Line{
					{Account: "5400", Debit: d.Amount},
					{Account: "1000", Credit: d.Amount},
				},
			}, nil
		}, ctx, r.ledger, e.ID)

	default:
		return nil
	}
}

// handle is a generic helper that unmarshals event data, builds a journal entry,
// and posts it to the ledger with causation metadata linking back to the source event.
func handle[T any](e fact.Event, build func(T) (JournalEntryPosted, error), ctx context.Context, ledger *Ledger, causationID string) error {
	var data T
	if err := json.Unmarshal(e.Data, &data); err != nil {
		return fmt.Errorf("unmarshal %s: %w", e.Type, err)
	}

	entry, err := build(data)
	if err != nil {
		return fmt.Errorf("build entry for %s: %w", e.Type, err)
	}

	if _, err := ledger.PostWithMetadata(ctx, entry, map[string]string{
		"causation_id": causationID,
	}); err != nil {
		return fmt.Errorf("post entry for %s: %w", e.Type, err)
	}

	return nil
}

func mustParseDate(s string) time.Time {
	t, err := time.Parse("2006-01-02", s)
	if err != nil {
		panic(fmt.Sprintf("invalid date %q: %v", s, err))
	}
	return t
}
