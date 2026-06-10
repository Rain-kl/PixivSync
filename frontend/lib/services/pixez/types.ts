export type PixezMirrorTarget = "illust" | "novel"

export type PixezMirrorStatusText = "success" | "processing" | "failed" | "none"

export type PixezRunStatus = "running" | "success" | "failed"

export interface PixezAccount {
  pixiv_user_id: string
  name: string
  account: string
  mail_address: string
  user_image: string
  is_premium: number
  x_restrict: number
  is_mail_authorized: number
  created_at: string
  updated_at: string
}

export interface PixezMirrorProgress {
  total: number
  succeeded: number
  processing: number
  failed: number
  not_queued: number
  percent: number
}

export interface PixezQueueStats {
  running: number
  queued: number
}

export interface PixezExportRun {
  id: string
  target_type: PixezMirrorTarget
  pixiv_user_id: string
  status: PixezRunStatus
  total_count: number
  new_count: number
  updated_count: number
  removed_count: number
  error_message: string
  started_at: string
  finished_at?: string | null
  duration_ms: number
  last_request_url: string
}

export interface PixezDashboard {
  accounts: number
  illusts: PixezMirrorProgress
  novels: PixezMirrorProgress
  queue: PixezQueueStats
  recent_runs: PixezExportRun[]
  updated_at: string
}

export interface PixezPaginatedResponse<T> {
  items: T[]
  total: number
  page: number
  page_size: number
}

export interface PixezBookmarkQuery {
  page?: number
  page_size?: number
  q?: string
  pixiv_user_id?: string
  mirror_status?: PixezMirrorStatusText | "all"
  work_status?: "active" | "visible" | "muted" | "unavailable" | "removed" | "all"
}

export interface PixezIllustBookmark {
  id: number
  pixiv_user_id: string
  restrict: string
  illust_id: number
  title: string
  type: string
  user_id: number
  user_name: string
  cover_url: string
  page_count: number
  width: number
  height: number
  sanity_level: number
  x_restrict: number
  total_bookmarks: number
  visible: boolean
  is_muted: boolean
  mirror_status: number
  mirror_status_text: PixezMirrorStatusText
  mirror_retry_count: number
  removed: boolean
  removed_at?: string | null
  last_seen_at: string
  updated_at: string
}

export interface PixezNovelBookmark {
  id: number
  pixiv_user_id: string
  restrict: string
  novel_id: number
  title: string
  caption: string
  user_id: number
  user_name: string
  cover_url: string
  text_length: number
  x_restrict: number
  total_bookmarks: number
  is_original: boolean
  visible: boolean
  is_muted: boolean
  series_id?: number | null
  series_title?: string | null
  mirror_status: number
  mirror_status_text: PixezMirrorStatusText
  mirror_retry_count: number
  removed: boolean
  removed_at?: string | null
  last_seen_at: string
  updated_at: string
}

export interface PixezMirrorImageFile {
  pixiv_url: string
  page: number
  upload_id: string
  file_name: string
  hash: string
  mime: string
  size: number
  storage_key: string
}

export interface PixezMirrorDetail {
  task_id: string
  status: string
  total_count: number
  success_count: number
  failed_count: number
  error_message: string
  created_at: string
  updated_at: string
}

export interface PixezIllustBookmarkDetail {
  item: PixezIllustBookmark
  mirror: PixezMirrorDetail | null
  image_files: PixezMirrorImageFile[]
  request_urls: string[]
  retry_urls: string[]
}

export interface PixezNovelBookmarkDetail {
  item: PixezNovelBookmark
  mirror: PixezMirrorDetail | null
  request_urls: string[]
  retry_urls: string[]
}

export interface PixezNovelTextPreview {
  text: string
}

export interface PixezMirrorStatus {
  task_id: string
  illust_id?: number
  novel_id?: number
  status: string
  mirrored: boolean
  total_count: number
  success_count: number
  failed_count: number
  request_urls_json?: string
  retry_urls_json?: string
  error_message?: string
}
