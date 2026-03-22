-- +goose Up
CREATE TABLE IF NOT EXISTS accounts (
    number     TEXT PRIMARY KEY,
    name       TEXT NOT NULL,
    type       TEXT NOT NULL CHECK (type IN ('asset', 'liability', 'equity', 'revenue', 'expense')),
    parent     TEXT REFERENCES accounts(number),
    active     BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL
);

-- +goose Down
DROP TABLE IF EXISTS accounts;
