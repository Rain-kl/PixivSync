import {useQuery} from "@tanstack/react-query"

import {PixezService} from "@/lib/services"
import type {
  PixezBookmarkQuery,
  PixezIllustBookmark,
  PixezMirrorTarget,
  PixezNovelBookmark,
  PixezPaginatedResponse,
} from "@/lib/services"

export type PixEzMirrorListItem = PixezIllustBookmark | PixezNovelBookmark
export type PixEzMirrorListResponse = PixezPaginatedResponse<PixEzMirrorListItem>

export function usePixEzMirrors(target: PixezMirrorTarget, params: PixezBookmarkQuery) {
  return useQuery<PixEzMirrorListResponse>({
    queryKey: ["pixez", "mirrors", target, params],
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
