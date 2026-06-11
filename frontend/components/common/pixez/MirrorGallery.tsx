"use client"

import {useState} from "react"
import Image from "next/image"
import {useQueryClient} from "@tanstack/react-query"
import {ChevronLeft, ChevronRight, Eye, ImageIcon} from "lucide-react"
import {toast} from "sonner"

import {Badge} from "@/components/ui/badge"
import {Button} from "@/components/ui/button"
import {Card, CardContent, CardFooter} from "@/components/ui/card"
import {EmptyStateWithBorder} from "@/components/layout/empty"
import {LoadingStateWithBorder} from "@/components/layout/loading"
import {PixezService} from "@/lib/services"
import type {PixezMirrorTarget, PixezMirroredIllust, PixezMirroredNovel} from "@/lib/services"

import {formatPixEzNumber, mirrorImageURL, pixezMirrorStatusLabel} from "./pixez-format"
import {MirrorDetailDrawer} from "./MirrorDetailDrawer"
import {MirrorPreviewDialog} from "./MirrorPreviewDialog"

type MirrorItem = PixezMirroredIllust | PixezMirroredNovel

function itemID(target: PixezMirrorTarget, item: MirrorItem) {
  return target === "illust"
    ? (item as PixezMirroredIllust).illust_id
    : (item as PixezMirroredNovel).novel_id
}

function itemMeta(target: PixezMirrorTarget, item: MirrorItem) {
  if (target === "illust") {
    const illust = item as PixezMirroredIllust
    return `${illust.page_count || 1} 页 · ${illust.width || "-"}x${illust.height || "-"}`
  }
  const novel = item as PixezMirroredNovel
  return `${formatPixEzNumber(novel.text_length)} 字${novel.series_title ? ` · ${novel.series_title}` : ""}`
}

function statusVariant(status: string) {
  if (status === "failed") return "destructive"
  if (status === "success") return "secondary"
  return "outline"
}

export function MirrorGallery({
  target,
  items,
  total,
  page,
  pageSize,
  loading,
  onPageChange,
}: {
  target: PixezMirrorTarget
  items: MirrorItem[]
  total: number
  page: number
  pageSize: number
  loading?: boolean
  onPageChange: (page: number) => void
}) {
  const queryClient = useQueryClient()
  const [selectedItem, setSelectedItem] = useState<MirrorItem | null>(null)
  const [detailOpen, setDetailOpen] = useState(false)
  const [previewItem, setPreviewItem] = useState<MirrorItem | null>(null)
  const [previewOpen, setPreviewOpen] = useState(false)
  const [retryingID, setRetryingID] = useState<number | null>(null)
  const totalPages = Math.max(1, Math.ceil(total / pageSize))

  const openDetail = (item: MirrorItem) => {
    setSelectedItem(item)
    setDetailOpen(true)
  }

  const openPreview = (item: MirrorItem) => {
    setPreviewItem(item)
    setPreviewOpen(true)
  }

  const handleRetry = async (item: MirrorItem) => {
    const id = itemID(target, item)
    try {
      setRetryingID(id)
      if (target === "illust") {
        await PixezService.mirrorIllust(id)
      } else {
        await PixezService.mirrorNovel(id)
      }
      toast.success("镜像任务已入队")
      await Promise.all([
        queryClient.invalidateQueries({queryKey: ["pixez", "dashboard"]}),
        queryClient.invalidateQueries({queryKey: ["pixez", "mirrors"]}),
        queryClient.invalidateQueries({queryKey: ["pixez", "mirror-detail", target, id]}),
      ])
    } catch (error) {
      toast.error("重新下载失败", {
        description: error instanceof Error ? error.message : "未知错误",
      })
    } finally {
      setRetryingID(null)
    }
  }

  if (loading && items.length === 0) {
    return <LoadingStateWithBorder icon={ImageIcon} description="加载镜像数据中..." />
  }

  if (items.length === 0) {
    return <EmptyStateWithBorder icon={ImageIcon} description="暂无镜像数据" />
  }

  return (
    <div className="flex flex-col gap-4">
      <div className="grid gap-3 sm:grid-cols-2 lg:grid-cols-3 2xl:grid-cols-4">
        {items.map((item) => {
          const id = itemID(target, item)
          const coverURL = mirrorImageURL(item.cover_url)
          return (
            <Card key={`${target}-${id}`} className="overflow-hidden rounded-lg py-0">
              <button
                type="button"
                className="group relative aspect-[4/3] w-full overflow-hidden bg-muted text-left"
                onClick={() => openDetail(item)}
              >
                {coverURL ? (
                  <Image
                    src={coverURL}
                    alt={item.title}
                    fill
                    unoptimized
                    sizes="(min-width: 1536px) 25vw, (min-width: 1024px) 33vw, (min-width: 640px) 50vw, 100vw"
                    className="object-cover transition-transform duration-300 group-hover:scale-[1.03]"
                  />
                ) : (
                  <div className="flex size-full items-center justify-center">
                    <ImageIcon className="text-muted-foreground" />
                  </div>
                )}
                <div className="absolute left-2 top-2 flex flex-wrap gap-1">
                  <Badge variant={statusVariant(item.status_text)}>
                    {pixezMirrorStatusLabel(item.status_text)}
                  </Badge>
                  {item.status_text === "failed" && (
                    <Badge variant="outline">失败 {item.failed_count}</Badge>
                  )}
                </div>
              </button>
              <CardContent className="flex flex-col gap-2 px-3 py-3">
                <div className="min-w-0">
                  <div className="truncate text-sm font-medium" title={item.title}>{item.title || `#${id}`}</div>
                  <div className="truncate text-xs text-muted-foreground" title={item.user_name}>
                    {item.user_name || "-"} · {item.user_id}
                  </div>
                </div>
                <div className="flex items-center justify-between gap-2 text-[11px] text-muted-foreground">
                  <span className="truncate">{itemMeta(target, item)}</span>
                  <Badge variant="outline">{item.success_count}/{item.total_count}</Badge>
                </div>
              </CardContent>
              <CardFooter className="flex gap-2 border-t px-3 py-2">
                <Button variant="outline" size="sm" className="flex-1" onClick={() => openDetail(item)}>
                  详情
                </Button>
                <Button variant="outline" size="sm" className="flex-1" onClick={() => openPreview(item)}>
                  <Eye data-icon="inline-start" />
                  预览
                </Button>
              </CardFooter>
            </Card>
          )
        })}
      </div>

      <div className="flex flex-col gap-2 border-t pt-3 sm:flex-row sm:items-center sm:justify-between">
        <div className="text-xs text-muted-foreground">
          共 {formatPixEzNumber(total)} 条，当前第 {page}/{totalPages} 页
        </div>
        <div className="flex items-center gap-2">
          <Button
            variant="outline"
            size="sm"
            onClick={() => onPageChange(Math.max(1, page - 1))}
            disabled={page <= 1 || loading}
          >
            <ChevronLeft />
            上一页
          </Button>
          <Button
            variant="outline"
            size="sm"
            onClick={() => onPageChange(Math.min(totalPages, page + 1))}
            disabled={page >= totalPages || loading}
          >
            下一页
            <ChevronRight />
          </Button>
        </div>
      </div>

      <MirrorDetailDrawer
        target={target}
        item={selectedItem}
        open={detailOpen}
        retrying={retryingID === (selectedItem ? itemID(target, selectedItem) : 0)}
        onOpenChange={setDetailOpen}
        onRetry={handleRetry}
      />
      <MirrorPreviewDialog
        target={target}
        item={previewItem}
        open={previewOpen}
        onOpenChange={setPreviewOpen}
      />
    </div>
  )
}
