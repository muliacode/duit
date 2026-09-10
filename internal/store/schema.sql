-- SQLite schema. Money is stored as INTEGER minor units (cents) — never
-- REAL/float — so arithmetic can't accumulate rounding error. Dates are
-- stored as ISO-8601 strings (RFC3339 for timestamps, YYYY-MM-DD for pure
-- dates) so they sort lexicographically and compare in plain SQL.

CREATE TABLE IF NOT EXISTS accounts (
                                        id                     TEXT PRIMARY KEY,
                                        name                   TEXT NOT NULL,
                                        type                   TEXT NOT NULL,
                                        institution            TEXT NOT NULL DEFAULT '',
                                        number                 TEXT NOT NULL DEFAULT '',
                                        currency               TEXT NOT NULL DEFAULT '',
                                        balance_minor          INTEGER NOT NULL DEFAULT 0,
                                        credit_limit_minor     INTEGER,
                                        original_amount_minor  INTEGER,
                                        interest_rate_bps      INTEGER,
                                        term_months            INTEGER,
                                        min_payment_minor      INTEGER,
                                        opened_at              TEXT,
                                        created_at             TEXT NOT NULL,
                                        updated_at             TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS transactions (
                                            id           TEXT PRIMARY KEY,
                                            account_id   TEXT NOT NULL REFERENCES accounts(id) ON DELETE CASCADE,
    date         TEXT NOT NULL,
    payee        TEXT NOT NULL,
    category     TEXT NOT NULL,
    amount_minor INTEGER NOT NULL,
    note         TEXT NOT NULL DEFAULT '',
    receipt_ref  TEXT NOT NULL DEFAULT '',
    created_at   TEXT NOT NULL,
    updated_at   TEXT NOT NULL
    );
CREATE INDEX IF NOT EXISTS idx_transactions_account  ON transactions(account_id);
CREATE INDEX IF NOT EXISTS idx_transactions_date     ON transactions(date);
CREATE INDEX IF NOT EXISTS idx_transactions_category ON transactions(category);
CREATE INDEX IF NOT EXISTS idx_transactions_payee    ON transactions(payee);

CREATE TABLE IF NOT EXISTS budget_categories (
                                                 id              TEXT PRIMARY KEY,
                                                 name            TEXT NOT NULL UNIQUE,
                                                 allocated_minor INTEGER NOT NULL DEFAULT 0,
                                                 created_at      TEXT NOT NULL,
                                                 updated_at      TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS bills (
                                     id               TEXT PRIMARY KEY,
                                     name             TEXT NOT NULL,
                                     amount_minor     INTEGER NOT NULL,
                                     cadence_interval INTEGER NOT NULL DEFAULT 1,
                                     cadence_unit     TEXT NOT NULL DEFAULT 'month',
                                     next_due         TEXT NOT NULL,
                                     account_id       TEXT NOT NULL REFERENCES accounts(id) ON DELETE CASCADE,
    status           TEXT NOT NULL DEFAULT 'upcoming',
    created_at       TEXT NOT NULL,
    updated_at       TEXT NOT NULL
    );
CREATE INDEX IF NOT EXISTS idx_bills_next_due ON bills(next_due);

CREATE TABLE IF NOT EXISTS goals (
                                     id                TEXT PRIMARY KEY,
                                     name              TEXT NOT NULL,
                                     target_minor      INTEGER NOT NULL,
                                     saved_minor       INTEGER NOT NULL DEFAULT 0,
                                     target_date       TEXT,
                                     linked_account_id TEXT REFERENCES accounts(id) ON DELETE SET NULL,
    created_at        TEXT NOT NULL,
    updated_at        TEXT NOT NULL
    );

-- Singleton settings row. The CHECK pins it to a single record.
CREATE TABLE IF NOT EXISTS settings (
                                        id               INTEGER PRIMARY KEY CHECK (id = 1),
    currency         TEXT NOT NULL DEFAULT 'EUR',
    locale           TEXT NOT NULL DEFAULT 'en',
    theme            TEXT NOT NULL DEFAULT 'system',
    date_format      TEXT NOT NULL DEFAULT 'human',
    hide_amounts     INTEGER NOT NULL DEFAULT 0,
    budget_method    TEXT NOT NULL DEFAULT 'simple',
    debt_strategy    TEXT NOT NULL DEFAULT 'avalanche',
    round_up_savings INTEGER NOT NULL DEFAULT 0,
    passcode_hash    TEXT NOT NULL DEFAULT ''
    );
INSERT OR IGNORE INTO settings (id) VALUES (1);

CREATE TABLE IF NOT EXISTS categorization_rules (
                                                    id         TEXT PRIMARY KEY,
                                                    match_text TEXT NOT NULL,
                                                    category   TEXT NOT NULL,
                                                    created_at TEXT NOT NULL
);
