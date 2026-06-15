-- +goose Up
INSERT INTO w_system_configs (key, value, type, visibility, description, created_at, updated_at)
VALUES (
    'push_config',
    '[]',
    'system',
    0,
    '通知推送渠道配置',
    CURRENT_TIMESTAMP,
    CURRENT_TIMESTAMP
) ON CONFLICT (key) DO NOTHING;

INSERT INTO w_push_events (event_key, name, channels, targets, template, enabled, created_at, updated_at)
VALUES (
    'admin_login',
    '管理员登录',
    '[]',
    '[]',
    '{"title": "管理员登录提醒", "content": "管理员 {{user.username}} 于 {{time}} 从 IP: {{ip}} 登录成功。", "level": "INFO"}',
    false,
    CURRENT_TIMESTAMP,
    CURRENT_TIMESTAMP
) ON CONFLICT (event_key) DO NOTHING;

-- +goose Down
DELETE FROM w_system_configs WHERE key = 'push_config';
DELETE FROM w_push_events WHERE event_key = 'admin_login';
