-- +goose Up
INSERT INTO w_schedules (name, task_type, cron, payload, is_active, created_at, updated_at)
SELECT '订阅每日同步检查', 'pixez_artist_due', '*/5 * * * *', '{}', true, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP
WHERE NOT EXISTS (SELECT 1 FROM w_schedules WHERE task_type = 'pixez_artist_due');

-- +goose Down
DELETE FROM w_schedules WHERE task_type = 'pixez_artist_due';
