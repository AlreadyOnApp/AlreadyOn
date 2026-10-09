-- +goose Up
-- Intel Quick Sync and Intel VA-API became one choice: the server uses Quick Sync when it works, else VA-API.
UPDATE server_settings SET transcoding = 'intel' WHERE transcoding IN ('intel-qsv', 'intel-vaapi');
-- +goose Down
UPDATE server_settings SET transcoding = 'intel-vaapi' WHERE transcoding = 'intel';
