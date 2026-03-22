# Lemonade Stand Story UI

**Date**: 2026-03-22
**Status**: Agreed

## Goal

A scroll-driven narrative UI that showcases axon-book by telling the story of a lemonade stand's first year. Demonstrates event-sourced double-entry bookkeeping with multi-stream event correlation.

## Architecture

```
Seed emits domain events              Projections build read models

operations stream:                     DailySummaryProjection
  sale.completed                         subscribes to: operations + ledger
  supply.purchased                       persists to: daily_summaries table
  spoilage.recorded
  permit.paid                          BalanceProjection (existing)
  investment.made                        subscribes to: ledger
         │                               in-memory (unchanged)
         ▼
    Reactor projector
    (translates domain → ledger)
         │
         ▼
ledger stream:                         UI reads from:
  journal_entry.posted                   - daily_summaries table
    metadata: {causation_id}             - trial balance / P&L (existing)
                                         - accounts (existing)
```

Domain events on the `operations` stream are translated into journal entries on the `ledger` stream by a reactor projector. Both appends commit atomically via transaction propagation.

## axon-fact evolution

### Transaction propagation

`PostgresStore.Append()` checks context for an existing `*sql.Tx`. If present, reuses it. If not, begins a new one. Enables projectors to call `Append()` on other streams within the same transaction.

`MemoryStore.Append()` updated for re-entrant locking to support the same pattern.

No new interfaces or methods — just smarter transaction handling inside `Append()`.

### Causation convention

Events link to their cause via metadata:

```go
Metadata: map[string]string{
    "causation_id": domainEvent.ID,
}
```

Convention established by axon-book, not enforced by axon-fact.

## Domain events

| Event | Stream | Data |
|-------|--------|------|
| `sale.completed` | operations | date, cups, price_per_cup, weather, temperature |
| `supply.purchased` | operations | date, items (name, quantity, unit_cost) |
| `spoilage.recorded` | operations | date, item, quantity, reason (rain/overripe) |
| `permit.paid` | operations | date, amount |
| `investment.made` | operations | date, amount, description |
| `journal_entry.posted` | ledger | (existing, unchanged) |

## Projections

### DailySummaryProjection (new, persisted)

Subscribes to both `operations` and `ledger` streams. Persists to `daily_summaries` table:

| Column | Source |
|--------|--------|
| date | domain events |
| cups_sold | sale.completed |
| price_per_cup | sale.completed |
| revenue | journal_entry.posted (account 4000) |
| weather | sale.completed |
| temperature | sale.completed |
| cogs_lemons | journal_entry.posted (account 5000) |
| cogs_sugar | journal_entry.posted (account 5100) |
| cogs_cups | journal_entry.posted (account 5200) |
| cogs_ice | journal_entry.posted (account 5300) |
| spoilage | journal_entry.posted (account 5600) |
| advertising | journal_entry.posted (account 5400) |
| permit | journal_entry.posted (account 5500) |

### BalanceProjection (existing, unchanged)

Filters on `journal_entry.posted` event type. Ignores operations events.

## API endpoints

### Existing (unchanged)
- `GET /api/accounts` — list accounts
- `GET /api/trial-balance` — current trial balance
- `GET /api/profit-and-loss?from=&to=` — P&L for date range

### New
- `GET /api/daily-summaries?from=&to=` — time-series data for charts
- `GET /api/monthly-summary` — aggregated by month for overview charts

## UI: scroll-driven narrative

SvelteKit app in `web/`, embedded via `//go:embed all:static`. LayerCake for charts.

### Chapters

| # | Title | Visualisation |
|---|-------|--------------|
| 1 | Opening Day | Chart of accounts appears, $5,000 investment animates into balance sheet |
| 2 | Spring Awakening | Daily revenue line chart (Mar-Apr), small numbers, muted |
| 3 | Summer Rush | Revenue scales up (Jun-Aug), cups sold stacks, weather icons overlay |
| 4 | The Cost of Lemonade | Stacked area of COGS components, ice spikes on hot days |
| 5 | When Life Gives You Spoilage | Spoilage events scattered on timeline, rainy days highlighted |
| 6 | Winding Down | Sep-Oct taper, mirror of spring |
| 7 | The Books | Full-year trial balance + P&L, everything balances, net income revealed |

Mini-nav with waypoints for jumping between chapters.

## Seed refactor

The seed command is refactored to emit domain events instead of constructing journal entries directly:

1. Emit `investment.made` → reactor posts opening journal entry
2. Emit `permit.paid` monthly → reactor posts expense entry
3. Emit `supply.purchased` weekly → reactor posts inventory + cash entry
4. Emit `sale.completed` daily → reactor posts revenue + COGS entry
5. Emit `spoilage.recorded` on spoilage days → reactor posts expense entry

The reactor projector contains the accounting translation logic: "a sale of N cups at $X produces a debit to cash, credit to revenue, debit to COGS, credit to inventory."

## Implementation order

1. axon-fact: transaction propagation (PostgresStore + MemoryStore)
2. axon-book: domain event types
3. axon-book: reactor projector (domain → ledger translation)
4. axon-book: seed refactor to emit domain events
5. axon-book: DailySummaryProjection + migration
6. axon-book: new API endpoints
7. axon-book: SvelteKit UI scaffolding + embed
8. axon-book: chapters 1-7 with LayerCake charts
