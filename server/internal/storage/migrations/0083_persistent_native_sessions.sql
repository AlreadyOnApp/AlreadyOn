-- +goose Up
-- Preserve currently valid native sessions without reviving expired or revoked credentials.
UPDATE browser_sessions SET expires_at_unix=253402300799
WHERE user_agent IN ('AlreadyOn-Android/TV','AlreadyOn-Android/Mobile')
AND expires_at_unix > CAST(strftime('%s','now') AS INTEGER);

-- +goose Down
-- A rollback cannot reconstruct individual original expiries; leave revocable sessions intact.
SELECT 1;
