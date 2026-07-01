-- +goose Up
ALTER TABLE urls ADD CONSTRAINT urls_unique_original UNIQUE (original);

-- +goose Down
ALTER TABLE urls DROP CONSTRAINT IF EXISTS urls_unique_original;
