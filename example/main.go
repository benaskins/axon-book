// Example composition root for axon-book general ledger service.
//
// Wires up:
//   - PostgreSQL (axon.OpenDB + migrations for both events and accounts tables)
//   - Event sourcing (fact.PostgresStore with balance projection + replay)
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
	fact "github.com/benaskins/axon-fact"
	"github.com/benaskins/axon-book/gl"
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
	db, err := axon.OpenDB(dsn, "book")
	if err != nil {
		return fmt.Errorf("open database: %w", err)
	}

	// Run migrations for the events table (from axon-fact)
	if err := axon.RunMigrations(db, fact.Migrations); err != nil {
		return fmt.Errorf("run event migrations: %w", err)
	}
	// Run migrations for the accounts table (from axon-book/gl)
	if err := axon.RunMigrations(db, gl.Migrations); err != nil {
		return fmt.Errorf("run gl migrations: %w", err)
	}

	// --- Domain ---
	projection := gl.NewBalanceProjection()
	events := fact.NewPostgresStore(db, fact.WithPgProjector(projection))
	accounts := gl.NewChartOfAccounts(db)
	ledger := gl.NewLedger(events, accounts, baseCurrency)

	// Replay persisted events to rebuild projections
	if err := events.Replay(ctx); err != nil {
		return fmt.Errorf("replay events: %w", err)
	}
	slog.Info("projections rebuilt", "currency", baseCurrency)

	// --- Auth ---
	authClient := axon.NewAuthClientPlain(authURL)
	requireAuth := axon.RequireAuth(authClient)

	// --- HTTP ---
	mux := http.NewServeMux()

	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	handler := gl.NewHandler(ledger, accounts, projection)
	handler.RegisterRoutes(mux, requireAuth)

	addr := ":" + port
	slog.Info("serving", "addr", addr, "auth_url", authURL)
	axon.ListenAndServe(addr, axon.StandardMiddleware(mux))
	return nil
}
