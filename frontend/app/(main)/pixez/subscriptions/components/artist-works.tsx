"use client"

import {useState} from "react"
import Image from "next/image"
import {useQuery} from "@tanstack/react-query"
import {ImageIcon} from "lucide-react"
import {ArtistService, PixezService} from "@/lib/services"
import type {PixezMirrorTarget, PixezMirrorImageFile} from "@/lib/services"
import type {ArtistWork} from "@/lib/services/pixez/artists.service"
import {Button} from "@/components/ui/button"
import {Badge} from "@/components/ui/badge"
import {Card, CardHeader, CardTitle, CardDescription, CardContent, CardFooter} from "@/components/ui/card"
import {Field, FieldLabel} from "@/components/ui/field"
import {Select, SelectContent, SelectGroup, SelectItem, SelectTrigger, SelectValue} from "@/components/ui/select"
import {Dialog, DialogContent, DialogHeader, DialogTitle, DialogDescription} from "@/components/ui/dialog"
import {ErrorInline} from "@/components/layout/error"
import {EmptyStateWithBorder} from "@/components/layout/empty"
import {LoadingStateWithBorder} from "@/components/layout/loading"
import {mirrorImageURL} from "@/components/common/pixez/pixez-format"
import {ArtistPagination} from "./subscription-list"

const statuses: Record<string, string> = {none: "未备份", complete: "完整备份", partial: "部分完成", failed: "失败", queued: "等待下载", processing: "下载中", success: "待检查完整性"}

function ArchivePreview({work, onClose}: {work: ArtistWork | null; onClose: () => void}) {
  const query = useQuery<{images?: PixezMirrorImageFile[]; text?: string}>({
    queryKey: ["artists", "archive", work?.target_type, work?.target_id],
    enabled: !!work,
    queryFn: async () => {
      if (!work) return {}
      if (work.target_type === "illust") {
        const detail = await PixezService.getMirroredIllustDetail(work.target_id)
        return {images: detail.image_files}
      }
      return PixezService.getMirroredNovelText(work.target_id)
    },
  })
  return <Dialog open={!!work} onOpenChange={open => {if (!open) onClose()}}>
    <DialogContent className="max-h-[90vh] overflow-y-auto sm:max-w-3xl">
      <DialogHeader><DialogTitle>{work?.title || "本地备份"}</DialogTitle>
        <DialogDescription>读取本地保存的内容，原站不可访问时仍可阅读。</DialogDescription></DialogHeader>
      {query.isLoading && <LoadingStateWithBorder />}
      {query.error && <ErrorInline error={query.error} onRetry={() => query.refetch()} />}
      {query.data?.text && <pre className="whitespace-pre-wrap break-words text-sm">{query.data.text}</pre>}
      {query.data?.images?.map(image => <Image key={image.upload_id} src={`/f/${image.upload_id}`} alt={`第 ${image.page + 1} 页`} width={1600} height={1200} unoptimized className="h-auto w-full object-contain" />)}
    </DialogContent>
  </Dialog>
}

export function ArtistWorks({id, target}: {id: string; target: PixezMirrorTarget}) {
  const [page, setPage] = useState(1)
  const [status, setStatus] = useState("all")
  const [selected, setSelected] = useState<ArtistWork | null>(null)
  const query = useQuery({queryKey: ["artists", "works", id, target, page, status], queryFn: () => ArtistService.works(id, target, page, status), enabled: !!id, refetchInterval: 10000})
  return <div className="flex flex-col gap-4">
    <Field className="sm:w-60"><FieldLabel>备份状态</FieldLabel>
      <Select value={status} onValueChange={value => {setStatus(value); setPage(1)}}>
        <SelectTrigger><SelectValue /></SelectTrigger><SelectContent><SelectGroup>
          <SelectItem value="all">全部作品</SelectItem><SelectItem value="complete">完整备份</SelectItem>
          <SelectItem value="none">未备份</SelectItem><SelectItem value="failed">失败或待补齐</SelectItem>
        </SelectGroup></SelectContent>
      </Select>
    </Field>
    {query.error && <ErrorInline error={query.error} onRetry={() => query.refetch()} />}
    {query.isLoading ? <LoadingStateWithBorder /> : query.data?.results.length === 0 ? <EmptyStateWithBorder title="暂无已发现作品" description="点击刷新目录，从 Pixiv 获取该作者当前可访问的作品。" /> :
      <div className="grid gap-4 sm:grid-cols-2 xl:grid-cols-4">{query.data?.results.map(work => <Card key={work.target_id}>
        <CardHeader><CardTitle className="truncate">{work.title || `作品 ${work.target_id}`}</CardTitle><CardDescription>ID {work.target_id}</CardDescription></CardHeader>
        <CardContent className="flex flex-col gap-3">
          {work.preview_url ? <Image src={mirrorImageURL(work.preview_url)} alt={work.title} width={400} height={280} unoptimized className="h-48 w-full rounded-md object-cover" /> : <ImageIcon className="size-12 text-muted-foreground" />}
          <Badge variant={work.backup_status === "failed" ? "destructive" : "secondary"}>{statuses[work.backup_status] || work.backup_status}</Badge>
          {work.not_returned && <p className="text-xs text-muted-foreground">最近扫描未返回；已有备份仍保留。</p>}
        </CardContent>
        <CardFooter><Button variant="outline" disabled={!["complete", "partial"].includes(work.backup_status)} onClick={() => setSelected(work)}>查看本地备份</Button></CardFooter>
      </Card>)}</div>}
    <ArtistPagination page={page} total={query.data?.total ?? 0} onChange={setPage} />
    <ArchivePreview work={selected} onClose={() => setSelected(null)} />
  </div>
}
