CREATE SCHEMA IF NOT EXISTS ledger;

GRANT USAGE ON SCHEMA ledger to default_user;
GRANT  CREATE ON SCHEMA ledger TO default_user;

ALTER DEFAULT PRIVILEGES IN SCHEMA ledger
GRANT ALL ON TABLE default_user;

CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
-- аккаунты - счета, т.к. логике не принципиальна какая-то метадата юзера, я решил не плодить
-- лишний мусор и оставить схему в формате счёт-транзакции и дополнительную таблицу для подсчёта результатов
CREATE TABLE ledger.accounts IF NOT EXISTS (
    user_id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    amount BIGINT NOT NULL DEFAULT 0 CHECK(amount >= 0),
    currency varchar(5) NOT NULL

);

CREATE TABLE IF NOT EXISTS ledger.transactions (
    id UUID PRIMARY KEY uuid_generate_v4(),
    user_from UUID FOREIGN KEY NOT NULL REFERENCES ledger.accounts(user_id),
    user_to UUID FOREIGN KEY NOT NULL REFERENCES ledger.accounts(user_id),
    currency SMALLINT NOT NULL,
    amount BIGINT NOT NULL CHECK (amount > 0),
    idempotency_key uuid NOT NULL UNIQUE,
    status SMALLINT NOT NULL DEFAULT 0,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT now()
    );

CREATE TABLE IF NOT EXISTS ledger.postings  (
    id BIGSERIAL PRIMARY KEY,
    transaction_id uuid NOT NULL REFERENCES ledger.transactions(id),
    account_id uuid NOT NULL REFERENCES ledger.accounts(user_id),
    amount BIGINT NOT NULL
    );

CREATE INDEX IF NOT EXISTS idx_postings_account_id ON ledger.postings(account_id);
CREATE INDEX IF NOT EXISTS idx_transactions_from_to ON ledger.transactions(user_from, user_to);