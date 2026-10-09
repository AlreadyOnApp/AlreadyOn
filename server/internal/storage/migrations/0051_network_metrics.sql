-- +goose Up
ALTER TABLE server_metrics ADD COLUMN ingress_mbps REAL;
ALTER TABLE server_metrics ADD COLUMN egress_mbps REAL;
ALTER TABLE server_metrics ADD COLUMN network_interface TEXT NOT NULL DEFAULT '';
ALTER TABLE server_metrics ADD COLUMN network_scope TEXT NOT NULL DEFAULT '';
-- +goose Down
ALTER TABLE server_metrics DROP COLUMN network_scope;
ALTER TABLE server_metrics DROP COLUMN network_interface;
ALTER TABLE server_metrics DROP COLUMN egress_mbps;
ALTER TABLE server_metrics DROP COLUMN ingress_mbps;
