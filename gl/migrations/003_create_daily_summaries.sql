-- +goose Up
CREATE TABLE IF NOT EXISTS daily_summaries (
    date         DATE PRIMARY KEY,
    cups_sold    INT NOT NULL DEFAULT 0,
    price_per_cup NUMERIC(10,2) NOT NULL DEFAULT 0,
    revenue      NUMERIC(10,2) NOT NULL DEFAULT 0,
    weather      TEXT NOT NULL DEFAULT '',
    cogs_lemons  NUMERIC(10,2) NOT NULL DEFAULT 0,
    cogs_sugar   NUMERIC(10,2) NOT NULL DEFAULT 0,
    cogs_cups    NUMERIC(10,2) NOT NULL DEFAULT 0,
    cogs_ice     NUMERIC(10,2) NOT NULL DEFAULT 0,
    spoilage     NUMERIC(10,2) NOT NULL DEFAULT 0,
    advertising  NUMERIC(10,2) NOT NULL DEFAULT 0,
    permit       NUMERIC(10,2) NOT NULL DEFAULT 0,
    ice_cost     NUMERIC(10,2) NOT NULL DEFAULT 0
);

-- +goose Down
DROP TABLE IF EXISTS daily_summaries;
