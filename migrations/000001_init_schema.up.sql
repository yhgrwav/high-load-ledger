CREATE SCHEMA IF NOT EXISTS ledger;

GRANT USAGE ON SCHEMA ledger TO default_user;
GRANT CREATE ON SCHEMA ledger TO default_user;
ALTER DEFAULT PRIVILEGES IN SCHEMA ledger GRANT ALL ON TABLES TO default_user;

CREATE TABLE IF NOT EXISTS ledger.accounts (
    user_id UUID PRIMARY KEY,
    amount BIGINT NOT NULL DEFAULT 0 CHECK(amount >= 0),
    currency SMALLINT NOT NULL,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT now()
    );

CREATE TABLE IF NOT EXISTS ledger.transactions (
    id UUID PRIMARY KEY,
    user_from UUID NOT NULL REFERENCES ledger.accounts(user_id),
    user_to UUID NOT NULL REFERENCES ledger.accounts(user_id),
    currency SMALLINT NOT NULL,
    amount BIGINT NOT NULL CHECK (amount > 0),
    idempotency_key UUID NOT NULL UNIQUE,
    status SMALLINT NOT NULL DEFAULT 0,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT now()
    );

CREATE TABLE IF NOT EXISTS ledger.postings (
    id BIGSERIAL PRIMARY KEY,
    transaction_id UUID NOT NULL REFERENCES ledger.transactions(id),
    account_id UUID NOT NULL REFERENCES ledger.accounts(user_id),
    amount BIGINT NOT NULL
    );

CREATE INDEX IF NOT EXISTS idx_postings_account_id ON ledger.postings(account_id);
CREATE INDEX IF NOT EXISTS idx_transactions_from_to ON ledger.transactions(user_from, user_to);