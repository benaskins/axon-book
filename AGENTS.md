# axon-book

Event-sourced double-entry bookkeeping — general ledger, chart of accounts, journal entries via axon-fact.

## Build & Test

```bash
go test ./...
go vet ./...
```

## Architecture

axon-book uses event sourcing (via axon-fact) for all ledger mutations. Commands emit domain events; projections materialise read models.

```
Command → Event → Reactor → Projection
                         → DailySummary
```

## Key Files

- `gl/ledger.go` — Ledger aggregate: validates and posts journal entries to event store
- `gl/entry.go` — JournalEntryPosted event data and Line type (debit/credit with decimal amounts)
- `gl/account.go` — Chart of accounts (CRUD, PostgreSQL-backed)
- `gl/events.go` — Domain event types (EntryPosted, etc.)
- `gl/reactor.go` — Event reactor: subscribes to events, drives projections
- `gl/projection.go` — Balance projection from event stream
- `gl/daily_summary.go` — DailySummaryProjection and summary API endpoints
- `gl/api.go` — REST API handlers (entries, accounts, balances, summaries)
- `gl/migrations.go` — Goose migration embed
- `embed.go` — Embedded SvelteKit UI static files
- `example/main.go` — Composition root wiring PostgreSQL, event store, projections, and HTTP

## Dependencies

- **axon** — HTTP server lifecycle, auth middleware
- **axon-fact** — Event, EventStore, Projector interfaces (PostgresStore)
- **shopspring/decimal** — Precise monetary arithmetic
