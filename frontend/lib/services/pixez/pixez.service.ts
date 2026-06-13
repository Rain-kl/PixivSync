import {BaseService} from "@/lib/services"
import type {
  PixezAccount,
  PixezBookmarkQuery,
  PixezDashboard,
  PixezExportRun,
  PixezIllustBookmark,
  PixezIllustBookmarkDetail,
  PixezMirrorStatus,
  PixezMirrorQuery,
  PixezMirroredIllust,
  PixezMirroredIllustDetail,
  PixezMirroredNovel,
  PixezMirroredNovelDetail,
  PixezNovelBookmark,
  PixezNovelBookmarkDetail,
  PixezNovelTextPreview,
  PixezPaginatedResponse,
  PixivProfileResponse,
} from "./types"

function cleanParams(params: PixezBookmarkQuery): Record<string, unknown> {
  const cleaned: Record<string, unknown> = {}

  Object.entries(params).forEach(([key, value]) => {
    if (value === undefined || value === "" || value === "all") return
    cleaned[key] = value
  })

  return cleaned
}

function cleanMirrorParams(params: PixezMirrorQuery): Record<string, unknown> {
  const cleaned: Record<string, unknown> = {}

  Object.entries(params).forEach(([key, value]) => {
    if (value === undefined || value === "" || value === "all") return
    cleaned[key] = value
  })

  return cleaned
}

export class PixezService extends BaseService {
  protected static readonly basePath = "/api/pixez"

  static async ping(): Promise<{ status: string }> {
    return this.get<{ status: string }>("/ping")
  }

  static async getDashboard(): Promise<PixezDashboard> {
    return this.get<PixezDashboard>("/dashboard")
  }

  static async listAccounts(): Promise<PixezAccount[]> {
    return this.get<PixezAccount[]>("/users")
  }

  static async addAccount(refreshToken: string): Promise<PixezAccount> {
    return this.post<PixezAccount>("/users", { refresh_token: refreshToken })
  }

  static async refreshAccountToken(pixivUserID: string): Promise<void> {
    return this.post<void>(`/users/${encodeURIComponent(pixivUserID)}/refresh-token`)
  }

  static async deleteAccount(pixivUserID: string): Promise<void> {
    return this.delete<void>(`/users/${encodeURIComponent(pixivUserID)}`)
  }

  static async listExportRuns(
    page = 1,
    pageSize = 20,
  ): Promise<PixezPaginatedResponse<PixezExportRun>> {
    return this.get<PixezPaginatedResponse<PixezExportRun>>("/bookmark-export-runs", {
      page,
      page_size: pageSize,
    })
  }

  static async listIllustBookmarks(
    params: PixezBookmarkQuery = {},
  ): Promise<PixezPaginatedResponse<PixezIllustBookmark>> {
    return this.get<PixezPaginatedResponse<PixezIllustBookmark>>(
      "/bookmarks/illusts",
      cleanParams(params),
    )
  }

  static async listNovelBookmarks(
    params: PixezBookmarkQuery = {},
  ): Promise<PixezPaginatedResponse<PixezNovelBookmark>> {
    return this.get<PixezPaginatedResponse<PixezNovelBookmark>>(
      "/bookmarks/novels",
      cleanParams(params),
    )
  }

  static async getIllustBookmarkDetail(illustID: number): Promise<PixezIllustBookmarkDetail> {
    return this.get<PixezIllustBookmarkDetail>(`/bookmarks/illusts/${illustID}/detail`)
  }

  static async getNovelBookmarkDetail(novelID: number): Promise<PixezNovelBookmarkDetail> {
    return this.get<PixezNovelBookmarkDetail>(`/bookmarks/novels/${novelID}/detail`)
  }

  static async getMirroredNovelText(novelID: number): Promise<PixezNovelTextPreview> {
    return this.rawGet<PixezNovelTextPreview>("/mirror/webview/v2/novel", {
      novel_id: novelID,
    })
  }

  static async listMirroredIllusts(
    params: PixezMirrorQuery = {},
  ): Promise<PixezPaginatedResponse<PixezMirroredIllust>> {
    return this.get<PixezPaginatedResponse<PixezMirroredIllust>>(
      "/mirror/illusts",
      cleanMirrorParams(params),
    )
  }

  static async listMirroredNovels(
    params: PixezMirrorQuery = {},
  ): Promise<PixezPaginatedResponse<PixezMirroredNovel>> {
    return this.get<PixezPaginatedResponse<PixezMirroredNovel>>(
      "/mirror/novels",
      cleanMirrorParams(params),
    )
  }

  static async getMirroredIllustDetail(illustID: number): Promise<PixezMirroredIllustDetail> {
    return this.get<PixezMirroredIllustDetail>(`/mirror/illusts/${illustID}/detail`)
  }

  static async getMirroredNovelDetail(novelID: number): Promise<PixezMirroredNovelDetail> {
    return this.get<PixezMirroredNovelDetail>(`/mirror/novels/${novelID}/detail`)
  }

  static async mirrorIllust(illustID: number): Promise<PixezMirrorStatus> {
    return this.post<PixezMirrorStatus>(`/illusts/${illustID}/mirror`)
  }

  static async mirrorNovel(novelID: number): Promise<PixezMirrorStatus> {
    return this.post<PixezMirrorStatus>(`/novels/${novelID}/mirror`)
  }

  static async getUserProfile(pixivUserID: string): Promise<PixivProfileResponse> {
    return this.get<PixivProfileResponse>(`/users/${encodeURIComponent(pixivUserID)}/profile`)
  }

  static async getLoginURL(): Promise<{ code_verifier: string; login_url: string }> {
    return this.get<{ code_verifier: string; login_url: string }>("/login-url")
  }

  static async loginCallback(code: string, codeVerifier: string): Promise<PixezAccount> {
    return this.post<PixezAccount>("/login-callback", { code, code_verifier: codeVerifier })
  }
}
