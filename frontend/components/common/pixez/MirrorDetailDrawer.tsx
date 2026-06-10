"use client"

import Link from "next/link"
import {ExternalLink, ImageIcon, RotateCcw} from "lucide-react"
import {useQuery} from "@tanstack/react-query"

import {Badge} from "@/components/ui/badge"
import {Button} from "@/components/ui/button"
import {Separator} from "@/components/ui/separator"
import {Sheet, SheetContent, SheetDescription, SheetFooter, SheetHeader, SheetTitle} from "@/components/ui/sheet"
import {Spinner} from "@/components/ui/spinner"
import {Table, TableBody, TableCell, TableHead, TableHeader, TableRow} from "@/components/ui/table"
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
  formatPixEzFileSize,
  mirrorImageURL,
  pixezMirrorStatusLabel,
  pixezTargetLabel,
} from "./pixez-format"

type MirrorItem = PixezIllustBookmark | PixezNovelBookmark
type MirrorDetail = PixezIllustBookmarkDetail | PixezNovelBookmarkDetail

function itemID(target: PixezMirrorTarget, item: MirrorItem | null) {
  if (!item) return 0
  return target === "illust"
    ? (item as PixezIllustBookmark).illust_id
    : (item as PixezNovelBookmark).novel_id
}

function originalURL(target: PixezMirrorTarget, id: number) {
  if (target === "illust") return `https://www.pixiv.net/artworks/${id}`
  return `https://www.pixiv.net/novel/show.php?id=${id}`
}

function detailTitle(target: PixezMirrorTarget, detail: MirrorDetail | undefined, item: MirrorItem | null) {
  if (detail) return detail.item.title
  if (!item) return pixezTargetLabel(target)
  return item.title
}

function detailAuthor(detail: MirrorDetail | undefined, item: MirrorItem | null) {
  if (detail) return detail.item.user_name
  return item?.user_name ?? "-"
}

function requestURLs(detail: MirrorDetail | undefined) {
  return detail?.request_urls ?? []
}

function retryURLs(detail: MirrorDetail | undefined) {
  return detail?.retry_urls ?? []
}

function isIllustDetail(detail: MirrorDetail | undefined): detail is PixezIllustBookmarkDetail {
  return !!detail && "image_files" in detail
}

function statusVariant(status: string | undefined) {
  if (status === "failed") return "destructive"
  if (status === "success") return "secondary"
  return "outline"
}

export function MirrorDetailDrawer({
  target,
  item,
  open,
  retrying,
  onOpenChange,
  onRetry,
}: {
  target: PixezMirrorTarget
  item: MirrorItem | null
  open: boolean
  retrying?: boolean
  onOpenChange: (open: boolean) => void
  onRetry: (item: MirrorItem) => Promise<void>
}) {
  const id = itemID(target, item)
  const detailQuery = useQuery<MirrorDetail>({
    queryKey: ["pixez", "mirror-detail", target, id],
    queryFn: () => target === "illust"
      ? PixezService.getIllustBookmarkDetail(id)
      : PixezService.getNovelBookmarkDetail(id),
    enabled: open && id > 0,
  })
  const detail = detailQuery.data
  const mirror = detail?.mirror

  return (
    <Sheet open={open} onOpenChange={onOpenChange}>
      <SheetContent className="w-full p-0 sm:max-w-[760px]">
        <SheetHeader className="border-b">
          <SheetTitle>{detailTitle(target, detail, item)}</SheetTitle>
          <SheetDescription>
            {pixezTargetLabel(target)} · {detailAuthor(detail, item)} · {id || "-"}
          </SheetDescription>
        </SheetHeader>

        <div className="flex-1 overflow-y-auto px-4 pb-4">
          {detailQuery.isLoading ? (
            <div className="py-4">
              <LoadingStateWithBorder icon={ImageIcon} description="加载镜像详情中..." />
            </div>
          ) : detailQuery.error ? (
            <div className="py-4">
              <ErrorInline error={detailQuery.error} onRetry={() => detailQuery.refetch()} />
            </div>
          ) : detail ? (
            <div className="flex flex-col gap-5 py-4">
              <div className="grid gap-3 sm:grid-cols-2">
                <div className="rounded-md border p-3">
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
                <div className="rounded-md border p-3">
                  <div className="text-xs text-muted-foreground">最近更新</div>
                  <div className="mt-2 font-mono text-xs">{formatPixEzDateTime(detail.item.updated_at)}</div>
                </div>
                <div className="rounded-md border p-3">
                  <div className="text-xs text-muted-foreground">画师 / 作者 ID</div>
                  <div className="mt-2 font-mono text-xs">{detail.item.user_id}</div>
                </div>
                <div className="rounded-md border p-3">
                  <div className="text-xs text-muted-foreground">分级</div>
                  <div className="mt-2 text-xs">
                    x_restrict={detail.item.x_restrict}
                    {"sanity_level" in detail.item ? ` · sanity=${detail.item.sanity_level}` : ""}
                  </div>
                </div>
              </div>

              <div className="flex flex-wrap gap-2">
                <Button variant="outline" size="sm" asChild>
                  <Link href={originalURL(target, id)} target="_blank" rel="noopener noreferrer">
                    <ExternalLink />
                    Pixiv 原链接
                  </Link>
                </Button>
                {mirror?.task_id && (
                  <Button variant="outline" size="sm" asChild>
                    <Link href="/admin/tasks/executions">
                      <ExternalLink />
                      任务日志
                    </Link>
                  </Button>
                )}
              </div>

              <Separator />

              {isIllustDetail(detail) && (
                <div className="flex flex-col gap-3">
                  <div className="text-sm font-medium">图片分页明细</div>
                  {detail.image_files.length === 0 ? (
                    <EmptyStateWithBorder description="暂无已保存图片页" />
                  ) : (
                    <div className="rounded-md border">
                      <Table className="min-w-[760px]">
                        <TableHeader>
                          <TableRow className="hover:bg-transparent">
                            <TableHead className="w-[60px]">页</TableHead>
                            <TableHead>Pixiv URL</TableHead>
                            <TableHead className="w-[150px]">存储 Key</TableHead>
                            <TableHead className="w-[90px]">大小</TableHead>
                            <TableHead className="w-[180px]">SHA-256</TableHead>
                          </TableRow>
                        </TableHeader>
                        <TableBody>
                          {detail.image_files.map((file) => (
                            <TableRow key={`${file.pixiv_url}-${file.page}`}>
                              <TableCell className="font-mono text-xs">{file.page + 1}</TableCell>
                              <TableCell className="max-w-[260px] truncate font-mono text-[11px]">
                                <Link href={mirrorImageURL(file.pixiv_url)} target="_blank" className="underline-offset-4 hover:underline">
                                  {file.pixiv_url}
                                </Link>
                              </TableCell>
                              <TableCell className="max-w-[150px] truncate font-mono text-[11px]">{file.storage_key}</TableCell>
                              <TableCell className="font-mono text-[11px]">{formatPixEzFileSize(file.size)}</TableCell>
                              <TableCell className="max-w-[180px] truncate font-mono text-[11px]">{file.hash}</TableCell>
                            </TableRow>
                          ))}
                        </TableBody>
                      </Table>
                    </div>
                  )}
                </div>
              )}

              <div className="grid gap-4 lg:grid-cols-2">
                <div className="flex flex-col gap-2">
                  <div className="text-sm font-medium">请求 URL</div>
                  <div className="min-h-28 rounded-md border bg-muted/30 p-3">
                    {requestURLs(detail).length === 0 ? (
                      <div className="text-xs text-muted-foreground">暂无记录</div>
                    ) : (
                      <div className="flex flex-col gap-2">
                        {requestURLs(detail).map((url) => (
                          <div key={url} className="break-all font-mono text-[11px] text-muted-foreground">{url}</div>
                        ))}
                      </div>
                    )}
                  </div>
                </div>
                <div className="flex flex-col gap-2">
                  <div className="text-sm font-medium">失败 URL</div>
                  <div className="min-h-28 rounded-md border bg-muted/30 p-3">
                    {retryURLs(detail).length === 0 ? (
                      <div className="text-xs text-muted-foreground">暂无记录</div>
                    ) : (
                      <div className="flex flex-col gap-2">
                        {retryURLs(detail).map((url) => (
                          <div key={url} className="break-all font-mono text-[11px] text-destructive">{url}</div>
                        ))}
                      </div>
                    )}
                  </div>
                </div>
              </div>

              {mirror?.error_message && (
                <div className="rounded-md border border-destructive/30 bg-destructive/5 p-3 text-sm text-destructive">
                  {mirror.error_message}
                </div>
              )}
            </div>
          ) : (
            <div className="py-4">
              <EmptyStateWithBorder description="未选择镜像记录" />
            </div>
          )}
        </div>

        <SheetFooter className="border-t">
          <Button variant="outline" onClick={() => detailQuery.refetch()} disabled={!item || detailQuery.isFetching}>
            {detailQuery.isFetching ? <Spinner /> : <RotateCcw />}
            刷新详情
          </Button>
          <Button onClick={() => item && onRetry(item)} disabled={!item || retrying}>
            {retrying ? <Spinner /> : <RotateCcw />}
            重新下载
          </Button>
        </SheetFooter>
      </SheetContent>
    </Sheet>
  )
}
