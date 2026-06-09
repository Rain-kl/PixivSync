-- +goose Up
CREATE TABLE IF NOT EXISTS pixiv_users (
    pixiv_user_id VARCHAR(64) PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    account VARCHAR(255) NOT NULL,
    mail_address VARCHAR(255),
    user_image VARCHAR(1024),
    access_token TEXT NOT NULL,
    refresh_token TEXT NOT NULL,
    device_token VARCHAR(255),
    is_premium INTEGER NOT NULL DEFAULT 0,
    x_restrict INTEGER NOT NULL DEFAULT 0,
    is_mail_authorized INTEGER NOT NULL DEFAULT 0,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX IF NOT EXISTS idx_pixiv_users_updated_at ON pixiv_users (updated_at);

CREATE TABLE IF NOT EXISTS ban_comments (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    pixiv_user_id VARCHAR(64) NOT NULL,
    comment_id VARCHAR(255) NOT NULL,
    name VARCHAR(255) NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_ban_comments_pixiv_user_id ON ban_comments (pixiv_user_id);

CREATE TABLE IF NOT EXISTS ban_illusts (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    pixiv_user_id VARCHAR(64) NOT NULL,
    illust_id VARCHAR(255) NOT NULL,
    name VARCHAR(255) NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_ban_illusts_pixiv_user_id ON ban_illusts (pixiv_user_id);

CREATE TABLE IF NOT EXISTS ban_tags (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    pixiv_user_id VARCHAR(64) NOT NULL,
    name VARCHAR(255) NOT NULL,
    translate_name VARCHAR(255) NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_ban_tags_pixiv_user_id ON ban_tags (pixiv_user_id);

CREATE TABLE IF NOT EXISTS ban_users (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    pixiv_user_id VARCHAR(64) NOT NULL,
    user_id VARCHAR(255) NOT NULL,
    name VARCHAR(255) NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_ban_users_pixiv_user_id ON ban_users (pixiv_user_id);

CREATE TABLE IF NOT EXISTS illust_histories (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    pixiv_user_id VARCHAR(64) NOT NULL,
    illust_id INTEGER NOT NULL,
    user_id INTEGER NOT NULL,
    picture_url TEXT NOT NULL,
    title VARCHAR(500),
    user_name VARCHAR(255),
    time BIGINT NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_illust_histories_pixiv_user_id ON illust_histories (pixiv_user_id);

CREATE TABLE IF NOT EXISTS novel_histories (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    pixiv_user_id VARCHAR(64) NOT NULL,
    novel_id INTEGER NOT NULL,
    user_id INTEGER NOT NULL,
    picture_url TEXT NOT NULL,
    title VARCHAR(500) NOT NULL,
    user_name VARCHAR(255) NOT NULL,
    time BIGINT NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_novel_histories_pixiv_user_id ON novel_histories (pixiv_user_id);

CREATE TABLE IF NOT EXISTS tag_histories (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    pixiv_user_id VARCHAR(64) NOT NULL,
    name VARCHAR(255) NOT NULL,
    translated_name VARCHAR(255) NOT NULL,
    type INTEGER DEFAULT 0
);
CREATE INDEX IF NOT EXISTS idx_tag_histories_pixiv_user_id ON tag_histories (pixiv_user_id);

-- +goose Down
DROP TABLE IF EXISTS tag_histories;
DROP TABLE IF EXISTS novel_histories;
DROP TABLE IF EXISTS illust_histories;
DROP TABLE IF EXISTS ban_users;
DROP TABLE IF EXISTS ban_tags;
DROP TABLE IF EXISTS ban_illusts;
DROP TABLE IF EXISTS ban_comments;
DROP TABLE IF EXISTS pixiv_users;
