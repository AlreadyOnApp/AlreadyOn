-- +goose Up
ALTER TABLE acme_tls ADD COLUMN agreement_accepted INTEGER NOT NULL DEFAULT 0;
ALTER TABLE acme_tls ADD COLUMN active INTEGER NOT NULL DEFAULT 0;
ALTER TABLE acme_tls ADD COLUMN configured INTEGER NOT NULL DEFAULT 0;
-- Earlier orders required explicit agreement, including staging orders.
UPDATE acme_tls SET agreement_accepted=(last_attempt>0),active=enabled,configured=(last_attempt>0);
-- +goose Down
ALTER TABLE acme_tls DROP COLUMN configured;
ALTER TABLE acme_tls DROP COLUMN active;
ALTER TABLE acme_tls DROP COLUMN agreement_accepted;
