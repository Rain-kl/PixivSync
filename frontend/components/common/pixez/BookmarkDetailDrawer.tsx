"use client"

import Image from "next/image"
import Link from "next/link"
import {ExternalLink, ImageIcon, RotateCcw} from "lucide-react"
import {useQuery} from "@tanstack/react-query"

import {Badge} from "@/components/ui/badge"
import {Button} from "@/components/ui/button"
import {Separator} from "@/components/ui/separator"
import {Sheet, SheetContent, SheetDescription, SheetFooter, SheetHeader, SheetTitle} from "@/components/ui/sheet"
import {Spinner} from "@/components/ui/spinner"
import {EmptyStateWithBorder} from "@/components/layout/empty"
import {ErrorInline} from "@/components/layout/error"
import {LoadingStateWithBorder} from "@/components/layout/loading"
import {PixezService} from "@/lib/services"
import type {
  PixezIllustBookmark,
  PixezIllustBookmarkDetail,
  PixezMirrorTarget,
  PixezNovelBookmark,
  PixezNovelBookmarkDetail,
} from "@/lib/services"

import {
  formatPixEzDateTime,
  formatPixEzNumber,
  mirrorImageURL,
  pixezMirrorStatusLabel,
  pixezTargetLabel,
} from "./pixez-format"

type BookmarkItem = PixezIllustBookmark | PixezNovelBookmark
type BookmarkDetail = PixezIllustBookmarkDetail | PixezNovelBookmarkDetail

interface OriginalTag {
  name?: string
  translated_name?: string | null
}

interface OriginalPayload {
  caption?: string
  height?: number
  image_urls?: {
    large?: string
    medium?: string
  }
  meta_pages?: Array<{
    image_urls?: {
      large?: string
      original?: string
    }
  }>
  page_count?: number
  tags?: OriginalTag[]
  text_length?: number
  total_bookmarks?: number
  total_view?: number
  width?: number
}

function parseOriginalPayload(value: unknown): OriginalPayload | null {
  if (!value) return null
  if (typeof value === "string") {
    try {
      const parsed: unknown = JSON.parse(value)
      return typeof parsed === "object" && parsed !== null ? parsed as OriginalPayload : null
    } catch {
      return null
    }
  }
  return typeof value === "object" ? value as OriginalPayload : null
}

function itemID(target: PixezMirrorTarget, item: BookmarkItem | null) {
  if (!item) return 0
  return target === "illust"
    ? (item as PixezIllustBookmark).illust_id
    : (item as PixezNovelBookmark).novel_id
}

function originalURL(target: PixezMirrorTarget, id: number) {
  if (target === "illust") return `https://www.pixiv.net/artworks/${id}`
  return `https://www.pixiv.net/novel/show.php?id=${id}`
}

function detailTitle(target: PixezMirrorTarget, detail: BookmarkDetail | undefined, item: BookmarkItem | null) {
  if (detail) return detail.item.title
  if (!item) return pixezTargetLabel(target)
  return item.title
}

function detailAuthor(detail: BookmarkDetail | undefined, item: BookmarkItem | null) {
  if (detail) return detail.item.user_name
  return item?.user_name ?? "-"
}

function statusVariant(status: string | undefined) {
  if (status === "failed") return "destructive"
  if (status === "success") return "secondary"
  return "outline"
}

export function BookmarkDetailDrawer({
  target,
  item,
  open,
  retrying,
  onOpenChange,
  onRetry,
}: {
  target: PixezMirrorTarget
  item: BookmarkItem | null
  open: boolean
  retrying?: boolean
  onOpenChange: (open: boolean) => void
  onRetry: (item: BookmarkItem) => Promise<void>
}) {
  const id = itemID(target, item)
  const detailQuery = useQuery<BookmarkDetail>({
    queryKey: ["pixez", "bookmark-detail-drawer", target, id],
    queryFn: () => target === "illust"
      ? PixezService.getIllustBookmarkDetail(id)
      : PixezService.getNovelBookmarkDetail(id),
    enabled: open && id > 0,
  })
  const detail = detailQuery.data
  const mirror = detail?.mirror

  // Extract original details from raw JSON payload
  const originalIllust = target === "illust" && detail && "illust_json" in detail && detail.illust_json
    ? parseOriginalPayload(detail.illust_json)
    : null

  const originalNovel = target === "novel" && detail && "novel_json" in detail && detail.novel_json
    ? parseOriginalPayload(detail.novel_json)
    : null

  // Get preview images: large preview preferred, fallback to medium
  const previewImages = (() => {
    if (!originalIllust) return []
    if (originalIllust.meta_pages && originalIllust.meta_pages.length > 0) {
      return originalIllust.meta_pages
        .map((page) => page.image_urls?.large || page.image_urls?.original || "")
        .filter(Boolean)
    }
    const previewURL = originalIllust.image_urls?.large || originalIllust.image_urls?.medium
    return previewURL ? [previewURL] : []
  })()

  // Get description/caption safely
  const caption = originalIllust?.caption || originalNovel?.caption || ""

  // Get original tags
  const tags = originalIllust?.tags || originalNovel?.tags || []

  return (
    <Sheet open={open} onOpenChange={onOpenChange}>
      <SheetContent className="w-full p-0 sm:max-w-[760px] flex flex-col h-full bg-background border-l">
        <SheetHeader className="border-b px-6 py-4 flex-none">
          <SheetTitle className="line-clamp-1">{detailTitle(target, detail, item)}</SheetTitle>
          <SheetDescription>
            {pixezTargetLabel(target)} · {detailAuthor(detail, item)} · {id || "-"}
          </SheetDescription>
        </SheetHeader>

        <div className="flex-1 overflow-y-auto px-6 pb-6">
          {detailQuery.isLoading ? (
            <div className="py-6">
              <LoadingStateWithBorder icon={ImageIcon} description="加载收藏详情中..." />
            </div>
          ) : detailQuery.error ? (
            <div className="py-6">
              <ErrorInline error={detailQuery.error} onRetry={() => detailQuery.refetch()} />
            </div>
          ) : detail ? (
            <div className="flex flex-col gap-6 py-4">
              {/* Basic Meta Cards */}
              <div className="grid gap-3 sm:grid-cols-2">
                <div className="rounded-md border p-3 bg-card/50">
                  <div className="text-xs text-muted-foreground">镜像状态</div>
                  <div className="mt-2 flex items-center gap-2">
                    <Badge variant={statusVariant(mirror?.status)}>
                      {pixezMirrorStatusLabel(mirror?.status || detail.item.mirror_status_text)}
                    </Badge>
                    {mirror && (
                      <span className="font-mono text-xs text-muted-foreground">
                        {mirror.success_count}/{mirror.total_count}
                      </span>
                    )}
                  </div>
                </div>
                <div className="rounded-md border p-3 bg-card/50">
                  <div className="text-xs text-muted-foreground">最后更新</div>
                  <div className="mt-2 font-mono text-xs">{formatPixEzDateTime(detail.item.updated_at)}</div>
                </div>
                <div className="rounded-md border p-3 bg-card/50">
                  <div className="text-xs text-muted-foreground">画师 / 作者 ID</div>
                  <div className="mt-2 font-mono text-xs">{detail.item.user_id}</div>
                </div>
                <div className="rounded-md border p-3 bg-card/50">
                  <div className="text-xs text-muted-foreground">分级</div>
                  <div className="mt-2 text-xs">
                    x_restrict={detail.item.x_restrict}
                    {"sanity_level" in detail.item ? ` · sanity=${detail.item.sanity_level}` : ""}
                  </div>
                </div>
              </div>

              {/* Action Link Buttons */}
              <div className="flex flex-wrap gap-2">
                <Button variant="outline" size="sm" asChild>
                  <Link href={originalURL(target, id)} target="_blank" rel="noopener noreferrer">
                    <ExternalLink className="mr-1 size-4" />
                    Pixiv 原链接
                  </Link>
                </Button>
                {mirror?.task_id && (
                  <Button variant="outline" size="sm" asChild>
                    <Link href="/admin/tasks?tab=executions">
                      <ExternalLink className="mr-1 size-4" />
                      任务日志
                    </Link>
                  </Button>
                )}
              </div>

              <Separator />

              {/* Original Info Section */}
              <div className="flex flex-col gap-4">
                <h3 className="text-sm font-semibold">原始作品信息</h3>
                
                {/* Stats */}
                <div className="flex flex-wrap gap-x-6 gap-y-2 text-xs text-muted-foreground">
                  <div>浏览数: <span className="font-mono font-medium text-foreground">{formatPixEzNumber(originalIllust?.total_view || originalNovel?.total_view)}</span></div>
                  <div>收藏数: <span className="font-mono font-medium text-foreground">{formatPixEzNumber(originalIllust?.total_bookmarks || originalNovel?.total_bookmarks)}</span></div>
                  {originalIllust && (
                    <>
                      <div>尺寸: <span className="font-mono font-medium text-foreground">{originalIllust.width} × {originalIllust.height}</span></div>
                      <div>图片页数: <span className="font-mono font-medium text-foreground">{originalIllust.page_count}</span></div>
                    </>
                  )}
                  {originalNovel && (
                    <div>字数: <span className="font-mono font-medium text-foreground">{formatPixEzNumber(originalNovel.text_length)}</span></div>
                  )}
                </div>

                {/* Description/Caption */}
                {caption && (
                  <div className="flex flex-col gap-2">
                    <span className="text-xs font-medium text-muted-foreground">作品描述</span>
                    <div 
                      className="text-sm border rounded-md p-3 bg-muted/20 max-h-[160px] overflow-y-auto break-words leading-relaxed select-text"
                      dangerouslySetInnerHTML={{ __html: caption }}
                    />
                  </div>
                )}

                {/* Tags */}
                {tags.length > 0 && (
                  <div className="flex flex-col gap-2">
                    <span className="text-xs font-medium text-muted-foreground">作品标签</span>
                    <div className="flex flex-wrap gap-1.5">
                      {tags.map((tag, index) => (
                        <div key={index} className="inline-flex flex-col px-2 py-1 bg-muted/60 border rounded text-xs select-text">
                          <span className="font-medium">{tag.name}</span>
                          {tag.translated_name && (
                            <span className="text-[10px] text-muted-foreground mt-0.5">{tag.translated_name}</span>
                          )}
                        </div>
                      ))}
                    </div>
                  </div>
                )}
              </div>

              {/* Image Previews */}
              {target === "illust" && previewImages.length > 0 && (
                <>
                  <Separator />
                  <div className="flex flex-col gap-3">
                    <h3 className="text-sm font-semibold">图片预览 ({previewImages.length} 页)</h3>
                    <div className="grid gap-4 sm:grid-cols-2">
                      {previewImages.map((url: string, index: number) => (
                        <div key={index} className="relative overflow-hidden rounded-md border bg-muted aspect-[3/4] flex items-center justify-center group hover:border-primary/50 transition-colors">
                          <Image
                            src={mirrorImageURL(url)}
                            alt={`Page ${index + 1}`}
                            fill
                            unoptimized
                            sizes="(min-width: 640px) 50vw, 100vw"
                            className="object-contain pointer-events-none"
                          />
                          <span className="absolute bottom-2 right-2 bg-black/60 text-white px-2 py-0.5 rounded text-[11px] font-mono">
                            {index + 1} / {previewImages.length}
                          </span>
                        </div>
                      ))}
                    </div>
                  </div>
                </>
              )}
            </div>
          ) : (
            <div className="py-6">
              <EmptyStateWithBorder description="未选择收藏记录" />
            </div>
          )}
        </div>

        <SheetFooter className="border-t px-6 py-4 flex-none gap-2 sm:gap-0">
          <Button variant="outline" onClick={() => detailQuery.refetch()} disabled={!item || detailQuery.isFetching}>
            {detailQuery.isFetching ? <Spinner /> : <RotateCcw />}
            刷新详情
          </Button>
          <Button onClick={() => item && onRetry(item)} disabled={!item || retrying}>
            {retrying ? <Spinner /> : <RotateCcw />}
            启动镜像
          </Button>
        </SheetFooter>
      </SheetContent>
    </Sheet>
  )
}
