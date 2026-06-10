-- +goose Up
CREATE TABLE IF NOT EXISTS mirror_illust (
    illust_id INTEGER PRIMARY KEY,
    task_id VARCHAR(128),
    status VARCHAR(32) NOT NULL DEFAULT 'queued',
    detail_json TEXT,
    image_files_json TEXT NOT NULL DEFAULT '[]',
    request_urls_json TEXT NOT NULL DEFAULT '[]',
    retry_urls_json TEXT NOT NULL DEFAULT '[]',
    error_message TEXT,
    total_count INTEGER NOT NULL DEFAULT 0,
    success_count INTEGER NOT NULL DEFAULT 0,
    failed_count INTEGER NOT NULL DEFAULT 0,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX IF NOT EXISTS idx_mirror_illust_task_id ON mirror_illust (task_id);
CREATE INDEX IF NOT EXISTS idx_mirror_illust_status ON mirror_illust (status);
CREATE INDEX IF NOT EXISTS idx_mirror_illust_updated_at ON mirror_illust (updated_at);

CREATE TABLE IF NOT EXISTS mirror_novel (
    novel_id INTEGER PRIMARY KEY,
    task_id VARCHAR(128),
    status VARCHAR(32) NOT NULL DEFAULT 'queued',
    detail_json TEXT,
    text_json TEXT,
    request_urls_json TEXT NOT NULL DEFAULT '[]',
    retry_urls_json TEXT NOT NULL DEFAULT '[]',
    error_message TEXT,
    total_count INTEGER NOT NULL DEFAULT 0,
    success_count INTEGER NOT NULL DEFAULT 0,
    failed_count INTEGER NOT NULL DEFAULT 0,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX IF NOT EXISTS idx_mirror_novel_task_id ON mirror_novel (task_id);
CREATE INDEX IF NOT EXISTS idx_mirror_novel_status ON mirror_novel (status);
CREATE INDEX IF NOT EXISTS idx_mirror_novel_updated_at ON mirror_novel (updated_at);

CREATE TABLE IF NOT EXISTS bookmark_export_runs (
    id VARCHAR(128) PRIMARY KEY,
    target_type VARCHAR(32) NOT NULL,
    pixiv_user_id VARCHAR(64) NOT NULL,
    restrict VARCHAR(32) NOT NULL,
    status VARCHAR(32) NOT NULL,
    total_count INTEGER NOT NULL DEFAULT 0,
    new_count INTEGER NOT NULL DEFAULT 0,
    updated_count INTEGER NOT NULL DEFAULT 0,
    removed_count INTEGER NOT NULL DEFAULT 0,
    error_message TEXT,
    started_at DATETIME NOT NULL,
    finished_at DATETIME,
    next_url TEXT,
    last_request_url TEXT,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX IF NOT EXISTS idx_bookmark_export_runs_target_user ON bookmark_export_runs (target_type, pixiv_user_id);
CREATE INDEX IF NOT EXISTS idx_bookmark_export_runs_status ON bookmark_export_runs (status);
CREATE INDEX IF NOT EXISTS idx_bookmark_export_runs_started_at ON bookmark_export_runs (started_at);

CREATE TABLE IF NOT EXISTS bookmark_illusts (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    pixiv_user_id VARCHAR(64) NOT NULL,
    restrict VARCHAR(32) NOT NULL,
    illust_id INTEGER NOT NULL,
    title TEXT,
    type VARCHAR(64),
    user_id INTEGER,
    user_name TEXT,
    page_count INTEGER,
    width INTEGER,
    height INTEGER,
    sanity_level INTEGER,
    x_restrict INTEGER,
    total_view INTEGER,
    total_bookmarks INTEGER,
    visible INTEGER NOT NULL DEFAULT 0,
    is_muted INTEGER NOT NULL DEFAULT 0,
    illust_ai_type INTEGER,
    illust_json TEXT NOT NULL,
    last_export_run_id VARCHAR(128) NOT NULL,
    last_seen_at DATETIME NOT NULL,
    mirror_status INTEGER NOT NULL DEFAULT 0,
    mirror_retry_count INTEGER NOT NULL DEFAULT 0,
    removed INTEGER NOT NULL DEFAULT 0,
    removed_at DATETIME,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(pixiv_user_id, restrict, illust_id)
);
CREATE INDEX IF NOT EXISTS idx_bookmark_illusts_user ON bookmark_illusts (pixiv_user_id);
CREATE INDEX IF NOT EXISTS idx_bookmark_illusts_removed ON bookmark_illusts (removed);
CREATE INDEX IF NOT EXISTS idx_bookmark_illusts_last_run ON bookmark_illusts (last_export_run_id);
CREATE INDEX IF NOT EXISTS idx_bookmark_illusts_mirror_status ON bookmark_illusts (mirror_status);

CREATE TABLE IF NOT EXISTS bookmark_novels (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    pixiv_user_id VARCHAR(64) NOT NULL,
    restrict VARCHAR(32) NOT NULL,
    novel_id INTEGER NOT NULL,
    title TEXT,
    caption TEXT,
    user_id INTEGER,
    user_name TEXT,
    text_length INTEGER,
    x_restrict INTEGER,
    total_view INTEGER,
    total_bookmarks INTEGER,
    is_original INTEGER NOT NULL DEFAULT 0,
    visible INTEGER NOT NULL DEFAULT 0,
    is_muted INTEGER NOT NULL DEFAULT 0,
    novel_ai_type INTEGER,
    series_id INTEGER,
    series_title TEXT,
    cover_url TEXT,
    novel_json TEXT NOT NULL,
    last_export_run_id VARCHAR(128) NOT NULL,
    last_seen_at DATETIME NOT NULL,
    mirror_status INTEGER NOT NULL DEFAULT 0,
    mirror_retry_count INTEGER NOT NULL DEFAULT 0,
    removed INTEGER NOT NULL DEFAULT 0,
    removed_at DATETIME,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(pixiv_user_id, restrict, novel_id)
);
CREATE INDEX IF NOT EXISTS idx_bookmark_novels_user ON bookmark_novels (pixiv_user_id);
CREATE INDEX IF NOT EXISTS idx_bookmark_novels_removed ON bookmark_novels (removed);
CREATE INDEX IF NOT EXISTS idx_bookmark_novels_last_run ON bookmark_novels (last_export_run_id);
CREATE INDEX IF NOT EXISTS idx_bookmark_novels_mirror_status ON bookmark_novels (mirror_status);

-- +goose Down
DROP TABLE IF EXISTS bookmark_novels;
DROP TABLE IF EXISTS bookmark_illusts;
DROP TABLE IF EXISTS bookmark_export_runs;
DROP TABLE IF EXISTS mirror_novel;
DROP TABLE IF EXISTS mirror_illust;
