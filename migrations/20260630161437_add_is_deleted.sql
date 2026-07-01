-- +goose Up
-- +goose StatementBegin
ALTER TABLE urls ADD COLUMN is_deleted BOOL NOT NULL DEFAULT false;

ALTER TABLE urls DROP CONSTRAINT IF EXISTS uq_urls_alias;
DROP INDEX IF EXISTS uidx_urls_alias;
CREATE UNIQUE INDEX uidx_urls_alias ON urls (alias) WHERE is_deleted = false;

ALTER TABLE urls DROP CONSTRAINT IF EXISTS uq_urls_original;
DROP INDEX IF EXISTS uidx_urls_original;
CREATE UNIQUE INDEX uidx_urls_original ON urls (original) WHERE is_deleted = false;

DROP INDEX IF EXISTS idx_urls_user_id;
CREATE INDEX idx_urls_user_id ON urls (user_id) WHERE is_deleted = false;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS idx_urls_user_id;
DROP INDEX IF EXISTS uidx_urls_original;
DROP INDEX IF EXISTS uidx_urls_alias;

DELETE FROM urls WHERE is_deleted = true;

ALTER TABLE urls ADD CONSTRAINT uq_urls_alias UNIQUE (alias);
ALTER TABLE urls ADD CONSTRAINT uq_urls_original UNIQUE (original);
CREATE INDEX idx_urls_user_id ON urls (user_id);

ALTER TABLE urls DROP COLUMN IF EXISTS is_deleted;
-- +goose StatementEnd
