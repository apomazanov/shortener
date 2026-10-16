-- +goose Up
-- +goose StatementBegin
CREATE MATERIALIZED VIEW stats AS
SELECT COUNT(DISTINCT user_id) AS users,
       COUNT(*) AS aliases,
       NOW() AS updated_at
FROM urls
WHERE is_deleted = false;

CREATE UNIQUE INDEX ux_stats_updated_at ON stats(updated_at);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP MATERIALIZED VIEW stats;
-- +goose StatementEnd
