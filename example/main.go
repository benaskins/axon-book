// Example composition root for axon-book general ledger service.
//
// Wires up:
//   - PostgreSQL (pool.NewPool + migrations for both events and accounts tables)
//   - Event sourcing (factpg.Store with balance projection + replay)
//   - Auth (axon.RequireAuth via axon-auth)
//   - REST API (gl.Handler.RegisterRoutes)
//   - Graceful shutdown (axon.ListenAndServe)
//
// Environment variables:
//
//	DATABASE_URL  — Postgres connection string (required)
//	AUTH_URL      — axon-auth service URL (required)
//	PORT          — listen port (default 8095)
//	BASE_CURRENCY — ledger base currency ISO 4217 (default AUD)
package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"

	"github.com/benaskins/axon"
	"github.com/benaskins/axon-base/migration"
	"github.com/benaskins/axon-base/pool"
	book "github.com/benaskins/axon-book"
	"github.com/benaskins/axon-book/gl"
	factpg "github.com/benaskins/axon-fact/postgres"
)

func main() {
	if err := run(); err != nil {
		slog.Error("fatal", "error", err)
		os.Exit(1)
	}
}

func run() error {
	ctx := context.Background()

	// --- Config ---
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		return fmt.Errorf("DATABASE_URL must be set")
	}
	authURL := os.Getenv("AUTH_URL")
	if authURL == "" {
		return fmt.Errorf("AUTH_URL must be set")
	}
	port := os.Getenv("PORT")
	if port == "" {
		port = "8095"
	}
	baseCurrency := os.Getenv("BASE_CURRENCY")
	if baseCurrency == "" {
		baseCurrency = "AUD"
	}

	// --- Database ---
	p, err := pool.NewPool(ctx, dsn, "book")
	if err != nil {
		return fmt.Errorf("open database: %w", err)
	}
	defer p.Close()
	db, err := p.StdDB()
	if err != nil {
		return fmt.Errorf("get sql.DB: %w", err)
	}

	// Run migrations for the events table (from axon-fact)
	if err := migration.Run(db, factpg.Migrations, "migrations"); err != nil {
		return fmt.Errorf("run event migrations: %w", err)
	}
	// Run migrations for the accounts table (from axon-book/gl)
	if err := migration.Run(db, gl.Migrations, "migrations"); err != nil {
		return fmt.Errorf("run gl migrations: %w", err)
	}

	// --- Domain ---
	projection := gl.NewBalanceProjection()
	dailySummaryProjection := gl.NewDailySummaryProjection(db)
	accounts := gl.NewChartOfAccounts(db)

	// Wire reactor: domain events on operations → journal entries on ledger.
	// The reactor, store, and ledger form a cycle; SetLedger breaks it.
	reactor := gl.NewReactor(nil)
	events := factpg.NewStore(db,
		factpg.WithProjector(reactor),
		factpg.WithProjector(projection),
		factpg.WithProjector(dailySummaryProjection),
	)
	ledger := gl.NewLedger(events, accounts, baseCurrency)
	reactor.SetLedger(ledger)

	// Replay persisted events to rebuild projections
	if err := events.Replay(ctx); err != nil {
		return fmt.Errorf("replay events: %w", err)
	}
	slog.Info("projections rebuilt", "currency", baseCurrency)

	// --- Auth ---
	authClient, err := axon.NewAuthClient(authURL)
	if err != nil {
		return fmt.Errorf("create auth client: %w", err)
	}
	requireAuth := axon.RequireAuth(authClient)

	// --- HTTP ---
	mux := http.NewServeMux()

	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	summaryStore := gl.NewDailySummaryStore(db)
	handler := gl.NewHandler(ledger, accounts, projection, summaryStore, events)
	handler.RegisterRoutes(mux, requireAuth)

	// Serve embedded SvelteKit UI
	mux.Handle("GET /", axon.SPAHandler(book.StaticFiles, "static", axon.WithStaticPrefix("/_app/")))

	slog.Info("serving", "port", port, "auth_url", authURL)
	axon.ListenAndServe(port, mux)
	return nil
}
