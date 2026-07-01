-- +goose Up
ALTER TABLE urls ADD COLUMN is_deleted BOOL NOT NULL DEFAULT false;

ALTER TABLE urls DROP CONSTRAINT IF EXISTS urls_alias_key;
CREATE UNIQUE INDEX idx_urls_unique_alias ON urls (alias) WHERE is_deleted = false;

ALTER TABLE urls DROP CONSTRAINT IF EXISTS urls_unique_original;
CREATE UNIQUE INDEX idx_urls_unique_original ON urls (original) WHERE is_deleted = false;

DROP INDEX IF EXISTS idx_urls_user_id;
CREATE INDEX idx_urls_user_id ON urls (user_id) WHERE is_deleted = false;

-- +goose Down
DROP INDEX IF EXISTS idx_urls_user_id;
DROP INDEX IF EXISTS idx_urls_unique_original;
DROP INDEX IF EXISTS idx_urls_unique_alias;

ALTER TABLE urls ADD CONSTRAINT idx_urls_unique_alias UNIQUE (alias);
ALTER TABLE urls ADD CONSTRAINT idx_urls_unique_original UNIQUE (original);
CREATE INDEX idx_urls_user_id ON urls (user_id);

ALTER TABLE urls DROP COLUMN IF EXISTS is_deleted;
