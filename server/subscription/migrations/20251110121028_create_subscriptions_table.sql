-- +goose Up
-- +goose StatementBegin
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

CREATE TABLE IF NOT EXISTS subscriptions (
  id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  service_name TEXT NOT NULL CHECK (length(trim(service_name)) > 0),
  price_rub INTEGER NOT NULL CHECK (price_rub >= 0),
  user_id UUID NOT NULL,
  start_month DATE NOT NULL,
  end_month DATE,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  CHECK (end_month IS NULL OR end_month >= start_month)
);

CREATE OR REPLACE FUNCTION set_updated_at()
RETURNS TRIGGER AS $$
BEGIN
  NEW.updated_at = now();
  RETURN NEW;
END; $$ LANGUAGE plpgsql;

CREATE TRIGGER subscriptions_set_updated_at
BEFORE UPDATE ON subscriptions
FOR EACH ROW EXECUTE PROCEDURE set_updated_at();
-- +goose StatementEnd


-- +goose Down
-- +goose StatementBegin
DROP TRIGGER IF EXISTS subscriptions_set_updated_at ON subscriptions;
DROP FUNCTION IF EXISTS set_updated_at;
DROP TABLE IF EXISTS subscriptions;
-- +goose StatementEnd
