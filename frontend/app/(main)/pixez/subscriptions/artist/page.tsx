"use client"

import {Suspense, useEffect, useRef, useState} from "react"
import Link from "next/link"
import {useSearchParams} from "next/navigation"
import {useMutation, useQuery, useQueryClient} from "@tanstack/react-query"
import {RefreshCw, UserRound} from "lucide-react"
import {toast} from "sonner"
import {ArtistService} from "@/lib/services"
import type {PixezMirrorTarget} from "@/lib/services"
import {Button} from "@/components/ui/button"
import {Tabs, TabsContent, TabsList, TabsTrigger} from "@/components/ui/tabs"
import {Card, CardHeader, CardTitle, CardDescription, CardContent} from "@/components/ui/card"
import {Badge} from "@/components/ui/badge"
import {ErrorInline} from "@/components/layout/error"
import {LoadingStateWithBorder} from "@/components/layout/loading"
import {ArtistWorks} from "../components/artist-works"
import {SubscriptionControls} from "../components/subscription-controls"

const scanLabels: Record<string, string> = {pending: "等待扫描", running: "扫描中", retrying: "等待重试", success: "扫描完成", failed: "扫描失败", cancelled: "已停止"}

function ArtistContent() {
  const params = useSearchParams()
  const id = params.get("artist_id") || ""
  const [target, setTarget] = useState<PixezMirrorTarget>(params.get("type") === "illust" ? "illust" : "novel")
  const validID = /^\d+$/.test(id)
  const client = useQueryClient()
  const refreshed = useRef(new Set<string>())
  const author = useQuery({queryKey: ["artists", "detail", id], queryFn: () => ArtistService.detail(id), enabled: validID, refetchInterval: 10000})
  const runs = useQuery({queryKey: ["artists", "runs", id], queryFn: () => ArtistService.runs(id), enabled: validID, refetchInterval: 5000})
  const refresh = useMutation({
    mutationFn: (type: PixezMirrorTarget) => ArtistService.refreshDirectory(id, type),
    onError: (error: Error) => toast.error(error.message),
    onSettled: () => client.invalidateQueries({queryKey: ["artists"]}),
  })
  const {mutate: refreshDirectory} = refresh
  useEffect(() => {
    if (!author.data || !runs.data) return
    const key = `${id}:${target}`
    if (refreshed.current.has(key)) return
    refreshed.current.add(key)
    const latest = runs.data.find(run => run.target_type === target)
    if (!latest || Date.now() - new Date(latest.created_at).getTime() > 24 * 60 * 60 * 1000) refreshDirectory(target)
  }, [author.data, runs.data, id, target, refreshDirectory])

  if (!validID) return <p className="py-6">作者 ID 无效。<Link href="/pixez/subscriptions">返回订阅管理</Link></p>
  if (author.isLoading) return <LoadingStateWithBorder />
  if (author.error) return <ErrorInline error={author.error} onRetry={() => author.refetch()} />
  const subscriptions = author.data?.subscriptions ?? []
  const current = subscriptions.find(sub => sub.target_type === target)
  return <div className="flex w-full flex-col gap-5 py-6">
    <Link href="/pixez/subscriptions" className="text-sm text-muted-foreground hover:underline">返回订阅管理</Link>
    <div className="flex flex-wrap items-center justify-between gap-3">
      <div className="flex items-center gap-2"><UserRound className="size-5 text-primary" /><h1 className="text-2xl font-semibold tracking-tight">{author.data?.artist.name || `作者 ${id}`}</h1></div>
      <Button variant="outline" disabled={refresh.isPending || !!current?.active_run_id} onClick={() => refreshDirectory(target)}><RefreshCw data-icon="inline-start" />刷新作品目录</Button>
    </div>
    <p className="text-sm text-muted-foreground">Pixiv ID {id} · 目录包含已发现的在线作品和本地历史备份。刷新目录不会开启订阅或下载原图、正文。</p>
    <div className="flex flex-wrap gap-6">
      <SubscriptionControls id={id} target="novel" subscription={subscriptions.find(sub => sub.target_type === "novel")} />
      <SubscriptionControls id={id} target="illust" subscription={subscriptions.find(sub => sub.target_type === "illust")} />
    </div>
    <Tabs value={target} onValueChange={value => setTarget(value as PixezMirrorTarget)}>
      <TabsList><TabsTrigger value="novel">小说作品</TabsTrigger><TabsTrigger value="illust">插画作品</TabsTrigger></TabsList>
      <TabsContent value="novel"><ArtistWorks id={id} target="novel" /></TabsContent>
      <TabsContent value="illust"><ArtistWorks id={id} target="illust" /></TabsContent>
    </Tabs>
    <Card>
      <CardHeader><CardTitle>最近扫描</CardTitle><CardDescription>作品扫描结束后，下载任务可能仍在进行；请以作品的备份状态为准。</CardDescription></CardHeader>
      <CardContent className="flex flex-col gap-3">
        {runs.error && <ErrorInline error={runs.error} onRetry={() => runs.refetch()} />}
        {runs.data?.length === 0 && <p className="text-sm text-muted-foreground">尚无扫描记录</p>}
        {runs.data?.slice(0, 10).map(run => <div key={run.id} className="flex flex-wrap items-center gap-3 text-sm">
          <Badge variant={run.status === "failed" ? "destructive" : "outline"}>{scanLabels[run.status] || run.status}</Badge>
          <span>{run.target_type === "novel" ? "小说" : "插画"} · {run.download ? "订阅同步" : "目录刷新"}</span>
          <span>发现 {run.discovered_count} · 安排下载 {run.queued_count} · 跳过 {run.skipped_count}</span>
          <span className="text-muted-foreground">{new Date(run.created_at).toLocaleString()}</span>
          {run.error_message && <span className="text-destructive">{run.error_message}</span>}
        </div>)}
      </CardContent>
    </Card>
  </div>
}

export default function ArtistPage() {
  return <Suspense fallback={<LoadingStateWithBorder />}><ArtistContent /></Suspense>
}
