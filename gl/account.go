package gl

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

// AccountType classifies an account for financial reporting.
type AccountType string

const (
	Asset     AccountType = "asset"
	Liability AccountType = "liability"
	Equity    AccountType = "equity"
	Revenue   AccountType = "revenue"
	Expense   AccountType = "expense"
)

// Account represents an entry in the chart of accounts.
type Account struct {
	Number    string      `json:"number"`
	Name      string      `json:"name"`
	Type      AccountType `json:"type"`
	Parent    string      `json:"parent,omitempty"`
	Active    bool        `json:"active"`
	CreatedAt time.Time   `json:"created_at"`
}

// ChartOfAccounts manages the chart of accounts in PostgreSQL.
type ChartOfAccounts struct {
	db *sql.DB
}

// NewChartOfAccounts creates a chart of accounts backed by the given database.
// The caller must run migrations before use.
func NewChartOfAccounts(db *sql.DB) *ChartOfAccounts {
	return &ChartOfAccounts{db: db}
}

// Create adds a new account to the chart.
func (c *ChartOfAccounts) Create(ctx context.Context, number, name string, typ AccountType, parent string) (*Account, error) {
	now := time.Now().UTC()
	_, err := c.db.ExecContext(ctx, `
		INSERT INTO accounts (number, name, type, parent, active, created_at)
		VALUES ($1, $2, $3, $4, true, $5)
	`, number, name, typ, nullString(parent), now)
	if err != nil {
		return nil, fmt.Errorf("insert account: %w", err)
	}
	return &Account{
		Number:    number,
		Name:      name,
		Type:      typ,
		Parent:    parent,
		Active:    true,
		CreatedAt: now,
	}, nil
}

// Get returns an account by number, or nil if not found.
func (c *ChartOfAccounts) Get(ctx context.Context, number string) (*Account, error) {
	var a Account
	var parent sql.NullString
	err := c.db.QueryRowContext(ctx, `
		SELECT number, name, type, parent, active, created_at
		FROM accounts WHERE number = $1
	`, number).Scan(&a.Number, &a.Name, &a.Type, &parent, &a.Active, &a.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("query account: %w", err)
	}
	a.Parent = parent.String
	return &a, nil
}

// List returns all active accounts ordered by number.
func (c *ChartOfAccounts) List(ctx context.Context) ([]Account, error) {
	rows, err := c.db.QueryContext(ctx, `
		SELECT number, name, type, parent, active, created_at
		FROM accounts WHERE active = true
		ORDER BY number
	`)
	if err != nil {
		return nil, fmt.Errorf("query accounts: %w", err)
	}
	defer rows.Close()

	var accounts []Account
	for rows.Next() {
		var a Account
		var parent sql.NullString
		if err := rows.Scan(&a.Number, &a.Name, &a.Type, &parent, &a.Active, &a.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan account: %w", err)
		}
		a.Parent = parent.String
		accounts = append(accounts, a)
	}
	return accounts, rows.Err()
}

// Deactivate marks an account as inactive.
func (c *ChartOfAccounts) Deactivate(ctx context.Context, number string) error {
	result, err := c.db.ExecContext(ctx, `
		UPDATE accounts SET active = false WHERE number = $1
	`, number)
	if err != nil {
		return fmt.Errorf("deactivate account: %w", err)
	}
	n, _ := result.RowsAffected()
	if n == 0 {
		return fmt.Errorf("account %s not found", number)
	}
	return nil
}

// Exists returns true if the account exists and is active.
func (c *ChartOfAccounts) Exists(ctx context.Context, number string) (bool, error) {
	var exists bool
	err := c.db.QueryRowContext(ctx, `
		SELECT EXISTS(SELECT 1 FROM accounts WHERE number = $1 AND active = true)
	`, number).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("check account exists: %w", err)
	}
	return exists, nil
}

func nullString(s string) sql.NullString {
	if s == "" {
		return sql.NullString{}
	}
	return sql.NullString{String: s, Valid: true}
}
