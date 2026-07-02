-- +goose Up
-- +goose StatementBegin
ALTER TABLE urls ADD CONSTRAINT uq_urls_original UNIQUE (original);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE urls DROP CONSTRAINT IF EXISTS uq_urls_original;
-- +goose StatementEnd
