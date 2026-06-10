-- +goose Up
INSERT INTO schedules (id, name, task_type, cron, payload, is_active, created_at, updated_at)
VALUES (2, 'PixEz 收藏自动入队镜像', 'pixez_auto_enqueue_bookmark_mirrors', '*/10 * * * *', '{}', TRUE, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
ON CONFLICT (id) DO NOTHING;

-- +goose Down
DELETE FROM schedules WHERE id = 2;
