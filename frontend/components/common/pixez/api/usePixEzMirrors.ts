import {useQuery} from "@tanstack/react-query"

import {PixezService} from "@/lib/services"
import type {
  PixezMirrorQuery,
  PixezMirrorTarget,
  PixezMirroredIllust,
  PixezMirroredNovel,
  PixezPaginatedResponse,
} from "@/lib/services"

export type PixEzMirrorListItem = PixezMirroredIllust | PixezMirroredNovel
export type PixEzMirrorListResponse = PixezPaginatedResponse<PixEzMirrorListItem>

export function usePixEzMirrors(target: PixezMirrorTarget, params: PixezMirrorQuery) {
  return useQuery<PixEzMirrorListResponse>({
    queryKey: ["pixez", "mirrors", target, params],
    queryFn: async () => {
      if (target === "illust") {
        const data = await PixezService.listMirroredIllusts(params)
        return {...data, items: data.items}
      }
      const data = await PixezService.listMirroredNovels(params)
      return {...data, items: data.items}
    },
  })
}
