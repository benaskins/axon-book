@AGENTS.md

## Conventions
- All ledger mutations are event-sourced via axon-fact: Command -> Event -> Reactor -> Projection
- Domain logic lives in `gl/` package  - Ledger aggregate validates and posts journal entries
- Uses shopspring/decimal for monetary arithmetic  - never use float64 for amounts
- Projections materialise read models (balances, daily summaries) from event streams
- Embedded SvelteKit UI via `//go:embed` in `embed.go`

## Constraints
- Double-entry invariant: every journal entry must balance (sum of debits == sum of credits)  - Ledger.Post enforces this
- Depends on axon and axon-fact  - do not add dependencies on other axon-* service packages
- Known divergence: `cmd/seed` installs to `~/.local/bin` but aurelia expects `${AURELIA_ROOT}/bin/book`
- Do not use float types for monetary values  - always shopspring/decimal

## Testing
- `go test ./...` runs all tests
- `go vet ./...` for lint
- Balance assertions should verify debit/credit equality after every journal entry
