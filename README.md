# axon-book

> Domain package · Part of the [lamina](https://github.com/benaskins/lamina-mono) workspace

Event-sourced double-entry bookkeeping — general ledger, chart of accounts, journal entries via axon-fact. All ledger mutations are captured as domain events; projections materialise read models for balances and daily summaries.

## Getting started

```
go get github.com/benaskins/axon-book@latest
```

axon-book is a domain package — it provides types, event handling, and HTTP handlers but no `main` function. You assemble it in your own composition root by wiring up PostgreSQL, an event store, and projections. See [`example/main.go`](example/main.go) for a complete wiring example.

```go
p, _ := pool.NewPool(ctx, dsn, "book")
db, _ := p.StdDB()
migration.Run(db, gl.Migrations, "migrations")

store := fact.NewPostgresStore(db)
projection := gl.NewBalanceProjection()
store.RegisterProjector(gl.StreamLedger, projection)
store.Replay(ctx)

accounts := gl.NewChartOfAccounts(db)
ledger := gl.NewLedger(store, accounts, "AUD")
handler := gl.NewHandler(ledger, accounts, projection, summaries, store)

mux := http.NewServeMux()
handler.RegisterRoutes(mux, axon.RequireAuth(authClient))
axon.ListenAndServe(port, mux)
```

## Key types

- **`gl.Ledger`** — General ledger aggregate: posts journal entries as events
- **`gl.JournalEntryPosted`**, **`gl.Line`** — Journal entry event data with debit/credit lines
- **`gl.ChartOfAccounts`** — Account CRUD backed by PostgreSQL
- **`gl.Account`**, **`gl.AccountType`** — Account types: asset, liability, equity, revenue, expense
- **`gl.BalanceProjection`** — Builds account balances from journal entry events, computes trial balance and P&L
- **`gl.DailySummaryProjection`** — Materialises daily operational summaries from event streams
- **`gl.Handler`** — REST API handlers for entries, accounts, balances, and summaries
- **`gl.Reactor`** — Event reactor that subscribes to operations events and posts journal entries

## License

MIT
