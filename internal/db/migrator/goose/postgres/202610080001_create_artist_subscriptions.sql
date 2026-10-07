-- +goose Up
CREATE TABLE pixez_artists (
 artist_id BIGINT PRIMARY KEY, name TEXT NOT NULL DEFAULT '', avatar_url TEXT NOT NULL DEFAULT '',
 created_at TIMESTAMPTZ, updated_at TIMESTAMPTZ
);
CREATE TABLE pixez_artist_works (
 id BIGSERIAL PRIMARY KEY, artist_id BIGINT NOT NULL, target_type TEXT NOT NULL, target_id BIGINT NOT NULL,
 first_summary_json TEXT NOT NULL DEFAULT '', discovered_from_mirror BOOLEAN NOT NULL DEFAULT false,
 discovered_from_scan BOOLEAN NOT NULL DEFAULT false, last_seen_run_id TEXT NOT NULL DEFAULT '',
 last_seen_at TIMESTAMPTZ, not_returned BOOLEAN NOT NULL DEFAULT false, created_at TIMESTAMPTZ, updated_at TIMESTAMPTZ
);
CREATE UNIQUE INDEX idx_artist_work_target ON pixez_artist_works(target_type, target_id);
CREATE INDEX idx_artist_work_list ON pixez_artist_works(artist_id, target_type, id);
CREATE TABLE pixez_artist_subscriptions (
 id BIGSERIAL PRIMARY KEY, artist_id BIGINT NOT NULL, target_type TEXT NOT NULL,
 enabled BOOLEAN NOT NULL DEFAULT false, revision BIGINT NOT NULL DEFAULT 0,
 active_run_id TEXT NOT NULL DEFAULT '', lease_expires_at TIMESTAMPTZ, next_due_at TIMESTAMPTZ,
 last_attempt_at TIMESTAMPTZ, last_success_at TIMESTAMPTZ, last_error TEXT NOT NULL DEFAULT '',
 created_at TIMESTAMPTZ, updated_at TIMESTAMPTZ
);
CREATE UNIQUE INDEX idx_artist_subscription_target ON pixez_artist_subscriptions(artist_id, target_type);
CREATE INDEX idx_artist_subscription_due ON pixez_artist_subscriptions(enabled, next_due_at);
CREATE TABLE pixez_artist_sync_runs (
 id TEXT PRIMARY KEY, execution_token TEXT NOT NULL DEFAULT '', lease_expires_at TIMESTAMPTZ, artist_id BIGINT NOT NULL, target_type TEXT NOT NULL, revision BIGINT NOT NULL DEFAULT 0,
 download BOOLEAN NOT NULL DEFAULT false, triggered_by TEXT NOT NULL DEFAULT '', account_id TEXT NOT NULL DEFAULT '',
 task_id TEXT NOT NULL DEFAULT '', status TEXT NOT NULL DEFAULT '', phase INTEGER NOT NULL DEFAULT 0,
 next_url TEXT NOT NULL DEFAULT '', discovered_count INTEGER NOT NULL DEFAULT 0,
 queued_count INTEGER NOT NULL DEFAULT 0, skipped_count INTEGER NOT NULL DEFAULT 0,
 error_message TEXT NOT NULL DEFAULT '', created_at TIMESTAMPTZ, updated_at TIMESTAMPTZ, finished_at TIMESTAMPTZ
);
CREATE INDEX idx_artist_run_list ON pixez_artist_sync_runs(artist_id, created_at);
ALTER TABLE mirror_illust ADD COLUMN execution_token TEXT NOT NULL DEFAULT '';
ALTER TABLE mirror_illust ADD COLUMN lease_expires_at TIMESTAMPTZ;
ALTER TABLE mirror_novel ADD COLUMN execution_token TEXT NOT NULL DEFAULT '';
ALTER TABLE mirror_novel ADD COLUMN lease_expires_at TIMESTAMPTZ;

-- +goose Down
ALTER TABLE mirror_novel DROP COLUMN lease_expires_at;
ALTER TABLE mirror_novel DROP COLUMN execution_token;
ALTER TABLE mirror_illust DROP COLUMN lease_expires_at;
ALTER TABLE mirror_illust DROP COLUMN execution_token;
DROP TABLE pixez_artist_sync_runs;
DROP TABLE pixez_artist_subscriptions;
DROP TABLE pixez_artist_works;
DROP TABLE pixez_artists;
