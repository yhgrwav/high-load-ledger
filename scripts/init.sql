CREATE USER default_user WITH PASSWORD 'password';
GRANT ALL PRIVILEGES ON DATABASE ledger TO default_user;
\c ledger;
GRANT ALL ON SCHEMA public TO default_user;