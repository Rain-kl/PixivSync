-- +goose Up
-- +goose StatementBegin
INSERT INTO w_schedules (name, task_type, cron, payload, is_active, created_at, updated_at)
SELECT 'PixEz 导出收藏', 'pixez_export_bookmarks', '0 0 * * *', '{}', 1, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP
WHERE NOT EXISTS (
    SELECT 1 FROM w_schedules WHERE task_type = 'pixez_export_bookmarks' AND name = 'PixEz 导出收藏'
);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DELETE FROM w_schedules WHERE task_type = 'pixez_export_bookmarks' AND name = 'PixEz 导出收藏';
-- +goose StatementEnd
