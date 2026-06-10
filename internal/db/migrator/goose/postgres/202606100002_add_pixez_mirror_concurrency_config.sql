-- +goose Up
INSERT INTO system_configs (key, value, type, visibility, description, created_at, updated_at) VALUES
    ('pixez_mirror_download_interval_seconds', '1', 'business', 1, 'Pixiv插画多图下载间隔（秒）', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
    ('pixez_mirror_illust_concurrency', '5', 'business', 1, 'Pixiv插画并发镜像限制（同时镜像的最大插画数）', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
    ('pixez_mirror_novel_concurrency', '5', 'business', 1, 'Pixiv小说并发镜像限制（同时镜像的最大小说数）', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
ON CONFLICT (key) DO NOTHING;

-- +goose Down
DELETE FROM system_configs WHERE key IN (
    'pixez_mirror_download_interval_seconds',
    'pixez_mirror_illust_concurrency',
    'pixez_mirror_novel_concurrency'
);
