-- +goose Up
CREATE UNIQUE INDEX idx_urls_unique_original ON urls(original);

-- +goose Down
DROP INDEX IF EXISTS idx_urls_unique_original;
