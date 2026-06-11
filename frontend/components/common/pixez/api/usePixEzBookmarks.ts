import {useQuery} from "@tanstack/react-query"

import {PixezService} from "@/lib/services"
import type {
  PixezBookmarkQuery,
  PixezIllustBookmark,
  PixezMirrorTarget,
  PixezNovelBookmark,
  PixezPaginatedResponse,
} from "@/lib/services"

export type PixEzBookmarkListItem = PixezIllustBookmark | PixezNovelBookmark
export type PixEzBookmarkListResponse = PixezPaginatedResponse<PixEzBookmarkListItem>

export function usePixEzBookmarks(target: PixezMirrorTarget, params: PixezBookmarkQuery) {
  return useQuery<PixEzBookmarkListResponse>({
    queryKey: ["pixez", "bookmarks", target, params],
    queryFn: async () => {
      if (target === "illust") {
        const data = await PixezService.listIllustBookmarks(params)
        return {...data, items: data.items}
      }
      const data = await PixezService.listNovelBookmarks(params)
      return {...data, items: data.items}
    },
  })
}
