"use client"

import Image from "next/image"
import Link from "next/link"
import {useQuery} from "@tanstack/react-query"
import {ExternalLink, FileText, ImageIcon, RefreshCw} from "lucide-react"

import {Button} from "@/components/ui/button"
import {Card, CardContent} from "@/components/ui/card"
import {
  Carousel,
  CarouselContent,
  CarouselItem,
  CarouselNext,
  CarouselPrevious,
} from "@/components/ui/carousel"
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog"
import {Separator} from "@/components/ui/separator"
import {Spinner} from "@/components/ui/spinner"
import {EmptyStateWithBorder} from "@/components/layout/empty"
import {ErrorInline} from "@/components/layout/error"
import {LoadingStateWithBorder} from "@/components/layout/loading"
import {PixezService} from "@/lib/services"
import type {
  PixezIllustBookmark,
  PixezIllustBookmarkDetail,
  PixezMirrorTarget,
  PixezMirroredIllust,
  PixezMirroredIllustDetail,
  PixezMirroredNovel,
  PixezMirroredNovelDetail,
  PixezNovelBookmark,
  PixezNovelBookmarkDetail,
  PixezNovelTextPreview,
} from "@/lib/services"

import {formatPixEzFileSize, formatPixEzNumber, mirrorImageURL, pixezTargetLabel} from "./pixez-format"

type MirrorItem =
  | PixezIllustBookmark
  | PixezNovelBookmark
  | PixezMirroredIllust
  | PixezMirroredNovel
type MirrorDetail =
  | PixezIllustBookmarkDetail
  | PixezNovelBookmarkDetail
  | PixezMirroredIllustDetail
  | PixezMirroredNovelDetail

function itemID(target: PixezMirrorTarget, item: MirrorItem | null) {
  if (!item) return 0
  return target === "illust"
    ? (item as PixezIllustBookmark | PixezMirroredIllust).illust_id
    : (item as PixezNovelBookmark | PixezMirroredNovel).novel_id
}

function isMirrorItem(item: MirrorItem | null) {
  return !!item && "status_text" in item
}

function previewTitle(target: PixezMirrorTarget, detail: MirrorDetail | undefined, item: MirrorItem | null) {
  if (detail?.item.title) return detail.item.title
  if (item?.title) return item.title
  const id = itemID(target, item)
  return id > 0 ? `${pixezTargetLabel(target)} #${id}` : `${pixezTargetLabel(target)}预览`
}

function previewAuthor(detail: MirrorDetail | undefined, item: MirrorItem | null) {
  if (detail?.item.user_name) return detail.item.user_name
  return item?.user_name || "-"
}

function originalURL(target: PixezMirrorTarget, id: number) {
  if (target === "illust") return `https://www.pixiv.net/artworks/${id}`
  return `https://www.pixiv.net/novel/show.php?id=${id}`
}

function isIllustDetail(
  detail: MirrorDetail | undefined,
): detail is PixezIllustBookmarkDetail | PixezMirroredIllustDetail {
  return !!detail && "image_files" in detail
}

function mirrorNovelTextURL(id: number) {
  return `/mirror/webview/v2/novel?novel_id=${id}`
}

function IllustPreview({
  detail,
}: {
  detail: PixezIllustBookmarkDetail | PixezMirroredIllustDetail
}) {
  if (detail.image_files.length === 0) {
    return <EmptyStateWithBorder icon={ImageIcon} description="暂无可预览的镜像图片页" />
  }

  return (
    <div className="flex flex-col gap-4">
      <Carousel className="mx-auto w-full max-w-[18rem] sm:max-w-md md:max-w-lg">
        <CarouselContent>
          {detail.image_files.map((file) => {
            const imageURL = mirrorImageURL(file.pixiv_url)
            return (
              <CarouselItem key={`${file.pixiv_url}-${file.page}`}>
                <div className="p-1">
                  <Card>
                    <CardContent className="flex aspect-square items-center justify-center p-2">
                      {imageURL ? (
                        <a
                          href={imageURL}
                          target="_blank"
                          rel="noopener noreferrer"
                          className="relative block size-full overflow-hidden rounded-md bg-muted"
                        >
                          <Image
                            src={imageURL}
                            alt={`${detail.item.title} ${file.page + 1}`}
                            fill
                            unoptimized
                            sizes="(min-width: 768px) 512px, 288px"
                            className="object-contain"
                          />
                        </a>
                      ) : (
                        <div className="flex size-full items-center justify-center rounded-md bg-muted">
                          <ImageIcon className="text-muted-foreground" />
                        </div>
                      )}
                    </CardContent>
                  </Card>
                </div>
              </CarouselItem>
            )
          })}
        </CarouselContent>
        <CarouselPrevious className="left-2" />
        <CarouselNext className="right-2" />
      </Carousel>

      <div className="grid gap-2 text-xs sm:grid-cols-3">
        <div className="rounded-md border p-3">
          <div className="text-muted-foreground">图片页数</div>
          <div className="mt-1 font-mono">{formatPixEzNumber(detail.image_files.length)}</div>
        </div>
        <div className="rounded-md border p-3">
          <div className="text-muted-foreground">镜像进度</div>
          <div className="mt-1 font-mono">
            {detail.mirror ? `${detail.mirror.success_count}/${detail.mirror.total_count}` : "-"}
          </div>
        </div>
        <div className="rounded-md border p-3">
          <div className="text-muted-foreground">已保存大小</div>
          <div className="mt-1 font-mono">
            {formatPixEzFileSize(detail.image_files.reduce((total, file) => total + file.size, 0))}
          </div>
        </div>
      </div>
    </div>
  )
}

function NovelPreview({
  detail,
  text,
  textLoading,
  textError,
  onRetryText,
}: {
  detail: PixezNovelBookmarkDetail | PixezMirroredNovelDetail
  text: PixezNovelTextPreview | undefined
  textLoading: boolean
  textError: Error | null
  onRetryText: () => void
}) {
  const coverURL = mirrorImageURL(detail.item.cover_url)
  const novelText = text?.text?.trim() ?? ""

  return (
    <div className="grid gap-4 lg:grid-cols-[260px_minmax(0,1fr)]">
      <div className="flex flex-col gap-3">
        <div className="relative aspect-[3/4] overflow-hidden rounded-md border bg-muted">
          {coverURL ? (
            <Image
              src={coverURL}
              alt={detail.item.title}
              fill
              unoptimized
              sizes="260px"
              className="object-cover"
            />
          ) : (
            <div className="flex size-full items-center justify-center">
              <FileText className="text-muted-foreground" />
            </div>
          )}
        </div>

        <div className="rounded-md border p-3 text-xs">
          <div className="flex items-center justify-between gap-3">
            <span className="text-muted-foreground">字数</span>
            <span className="font-mono">{formatPixEzNumber(detail.item.text_length)}</span>
          </div>
          <Separator className="my-2" />
          <div className="flex items-center justify-between gap-3">
            <span className="text-muted-foreground">收藏</span>
            <span className="font-mono">{formatPixEzNumber(detail.item.total_bookmarks)}</span>
          </div>
          {detail.item.series_title && (
            <>
              <Separator className="my-2" />
              <div className="flex flex-col gap-1">
                <span className="text-muted-foreground">系列</span>
                <span className="line-clamp-2">{detail.item.series_title}</span>
              </div>
            </>
          )}
        </div>
      </div>

      <div className="min-h-[420px] rounded-md border bg-muted/20 p-4">
        {textLoading ? (
          <LoadingStateWithBorder icon={FileText} description="加载小说镜像文本中..." />
        ) : textError ? (
          <ErrorInline error={textError} onRetry={onRetryText} />
        ) : novelText ? (
          <div className="max-h-[62vh] overflow-y-auto whitespace-pre-wrap break-words text-sm leading-7">
            {novelText}
          </div>
        ) : (
          <EmptyStateWithBorder icon={FileText} description="暂无可预览的小说正文" />
        )}
      </div>
    </div>
  )
}

export function MirrorPreviewDialog({
  target,
  item,
  open,
  onOpenChange,
}: {
  target: PixezMirrorTarget
  item: MirrorItem | null
  open: boolean
  onOpenChange: (open: boolean) => void
}) {
  const id = itemID(target, item)
  const mirrorItem = isMirrorItem(item)
  const detailQuery = useQuery<MirrorDetail>({
    queryKey: ["pixez", "mirror-preview-detail", target, id, mirrorItem],
    queryFn: () => {
      if (target === "illust") {
        return mirrorItem
          ? PixezService.getMirroredIllustDetail(id)
          : PixezService.getIllustBookmarkDetail(id)
      }
      return mirrorItem
        ? PixezService.getMirroredNovelDetail(id)
        : PixezService.getNovelBookmarkDetail(id)
    },
    enabled: open && id > 0,
  })
  const novelTextQuery = useQuery<PixezNovelTextPreview>({
    queryKey: ["pixez", "mirror-preview-novel-text", id],
    queryFn: () => PixezService.getMirroredNovelText(id),
    enabled: open && target === "novel" && id > 0,
  })
  const detail = detailQuery.data

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="gap-4 p-4 sm:max-w-[980px] md:max-w-[1040px]">
        <DialogHeader>
          <DialogTitle>{previewTitle(target, detail, item)}</DialogTitle>
          <DialogDescription>
            {pixezTargetLabel(target)} · {previewAuthor(detail, item)} · {id || "-"}
          </DialogDescription>
        </DialogHeader>

        {detailQuery.isLoading ? (
          <LoadingStateWithBorder icon={target === "illust" ? ImageIcon : FileText} description="加载预览数据中..." />
        ) : detailQuery.error ? (
          <ErrorInline error={detailQuery.error} onRetry={() => void detailQuery.refetch()} />
        ) : detail && isIllustDetail(detail) ? (
          <IllustPreview detail={detail} />
        ) : detail ? (
          <NovelPreview
            detail={detail}
            text={novelTextQuery.data}
            textLoading={novelTextQuery.isLoading}
            textError={novelTextQuery.error}
            onRetryText={() => void novelTextQuery.refetch()}
          />
        ) : (
          <EmptyStateWithBorder description="未选择镜像记录" />
        )}

        <DialogFooter>
          <Button
            variant="outline"
            onClick={() => {
              void detailQuery.refetch()
              if (target === "novel") {
                void novelTextQuery.refetch()
              }
            }}
            disabled={!item || detailQuery.isFetching || novelTextQuery.isFetching}
          >
            {detailQuery.isFetching || novelTextQuery.isFetching ? <Spinner /> : <RefreshCw data-icon="inline-start" />}
            刷新预览
          </Button>
          {id > 0 && (
            <>
              {target === "novel" && (
                <Button variant="outline" asChild>
                  <Link href={mirrorNovelTextURL(id)} target="_blank" rel="noopener noreferrer">
                    <ExternalLink data-icon="inline-start" />
                    镜像文本
                  </Link>
                </Button>
              )}
              <Button asChild>
                <Link href={originalURL(target, id)} target="_blank" rel="noopener noreferrer">
                  <ExternalLink data-icon="inline-start" />
                  Pixiv
                </Link>
              </Button>
            </>
          )}
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}
