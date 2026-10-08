import {BaseService} from "@/lib/services/core/base.service"
import type {PixezMirrorTarget} from "./types"

export interface ArtistSubscription {
  artist_id: string
  target_type: PixezMirrorTarget
  enabled: boolean
  active_run_id: string
  last_success_at: string | null
  next_due_at: string | null
  last_error: string
}
export interface Artist {
  artist_id: string
  name: string
  avatar_url: string
}
export interface ArtistListItem extends Artist {
  subscription: ArtistSubscription
  work_count: number
  backup_count: number
}
export interface ArtistWork {
  target_id: string
  target_type: PixezMirrorTarget
  title: string
  preview_url: string
  backup_status: string
  not_returned: boolean
}
export interface ArtistScanRun {
  id: string
  task_id: string
  target_type: PixezMirrorTarget
  status: string
  download: boolean
  discovered_count: number
  queued_count: number
  skipped_count: number
  error_message: string
  created_at: string
}
export interface ArtistPage<T> {total: number; results: T[]}

export class ArtistService extends BaseService {
  protected static readonly basePath = "/api/pixez/artists"

  static list(type: PixezMirrorTarget, page: number, q: string, subscription: string) {
    return this.get<ArtistPage<ArtistListItem>>("", {type, page, page_size: 24, q, subscription})
  }
  static detail(id: string) {
    return this.get<{artist: Artist; subscriptions: ArtistSubscription[]}>(`/${encodeURIComponent(id)}`)
  }
  static works(id: string, type: PixezMirrorTarget, page: number, status: string) {
    return this.get<ArtistPage<ArtistWork>>(`/${encodeURIComponent(id)}/works`, {type, page, page_size: 24, status})
  }
  static setSubscription(id: string, type: PixezMirrorTarget, enabled: boolean) {
    return this.put<ArtistSubscription[]>(`/${encodeURIComponent(id)}/subscriptions/${type}`, {enabled})
  }
  static sync(id: string, type: PixezMirrorTarget) {
    return this.post<ArtistScanRun>(`/${encodeURIComponent(id)}/subscriptions/${type}/sync`)
  }
  static refreshDirectory(id: string, type: PixezMirrorTarget) {
    return this.post<ArtistScanRun>(`/${encodeURIComponent(id)}/refresh?type=${type}`)
  }
  static runs(id: string) {
    return this.get<ArtistScanRun[]>(`/${encodeURIComponent(id)}/sync-runs`)
  }
}
