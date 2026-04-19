DROP INDEX IF EXISTS ledger.idx_transactions_from_to;
DROP INDEX IF EXISTS ledger.idx_postings_account_id;

DROP TABLE IF EXISTS ledger.postings;
DROP TABLE IF EXISTS ledger.transactions;
DROP TABLE IF EXISTS ledger.accounts;

DROP SCHEMA IF EXISTS ledger CASCADE;

DROP EXTENSION IF EXISTS "uuid-ossp";