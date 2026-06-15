-- +goose Up
CREATE TABLE w_push_channels (
    id BIGSERIAL PRIMARY KEY,
    name VARCHAR(80) NOT NULL UNIQUE,
    description VARCHAR(255),
    type VARCHAR(50) NOT NULL DEFAULT 'custom',
    token VARCHAR(100),
    url TEXT NOT NULL,
    other TEXT NOT NULL,
    enabled BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL
);

CREATE INDEX idx_w_push_channels_name ON w_push_channels(name);
CREATE INDEX idx_w_push_channels_enabled ON w_push_channels(enabled);

INSERT INTO w_system_configs (key, value, type, visibility, description, created_at, updated_at)
VALUES (
    'push_global_token',
    '',
    'system',
    0,
    '系统全局推送鉴权令牌',
    CURRENT_TIMESTAMP,
    CURRENT_TIMESTAMP
) ON CONFLICT (key) DO NOTHING;

-- +goose Down
DELETE FROM w_system_configs WHERE key = 'push_global_token';
DROP TABLE IF EXISTS w_push_channels;
