"use client"

import {useState} from "react"
import Link from "next/link"
import {useQuery} from "@tanstack/react-query"
import {RefreshCw} from "lucide-react"
import {ArtistService} from "@/lib/services"
import type {PixezMirrorTarget} from "@/lib/services"
import {Button} from "@/components/ui/button"
import {Input} from "@/components/ui/input"
import {Field, FieldGroup, FieldLabel} from "@/components/ui/field"
import {Select, SelectContent, SelectGroup, SelectItem, SelectTrigger, SelectValue} from "@/components/ui/select"
import {Avatar, AvatarFallback, AvatarImage} from "@/components/ui/avatar"
import {Badge} from "@/components/ui/badge"
import {Card, CardHeader, CardTitle, CardDescription, CardContent, CardFooter} from "@/components/ui/card"
import {EmptyStateWithBorder} from "@/components/layout/empty"
import {LoadingStateWithBorder} from "@/components/layout/loading"
import {ErrorInline} from "@/components/layout/error"
import {SubscriptionControls} from "./subscription-controls"

export function ArtistPagination({page, total, onChange}: {page: number; total: number; onChange: (page: number) => void}) {
  const pages = Math.max(1, Math.ceil(total / 24))
  return <div className="flex items-center justify-end gap-3">
    <span className="text-sm text-muted-foreground">共 {total} 条 · {page}/{pages}</span>
    <Button variant="outline" disabled={page <= 1} onClick={() => onChange(page - 1)}>上一页</Button>
    <Button variant="outline" disabled={page >= pages} onClick={() => onChange(page + 1)}>下一页</Button>
  </div>
}

export function SubscriptionList({target}: {target: PixezMirrorTarget}) {
  const [page, setPage] = useState(1)
  const [q, setQ] = useState("")
  const [filter, setFilter] = useState("all")
  const query = useQuery({
    queryKey: ["artists", "list", target, page, q, filter],
    queryFn: () => ArtistService.list(target, page, q, filter),
    refetchInterval: 15000,
  })
  return <div className="flex flex-col gap-4">
    <FieldGroup className="flex flex-col gap-3 sm:flex-row sm:items-end">
      <Field><FieldLabel htmlFor={`author-q-${target}`}>搜索作者</FieldLabel>
        <Input id={`author-q-${target}`} placeholder="作者名称、Pixiv ID" value={q} onChange={e => {setQ(e.target.value); setPage(1)}} />
      </Field>
      <Field><FieldLabel>订阅状态</FieldLabel>
        <Select value={filter} onValueChange={value => {setFilter(value); setPage(1)}}>
          <SelectTrigger><SelectValue /></SelectTrigger>
          <SelectContent><SelectGroup>
            <SelectItem value="all">全部作者</SelectItem><SelectItem value="enabled">已订阅</SelectItem>
            <SelectItem value="disabled">未订阅</SelectItem><SelectItem value="error">同步异常</SelectItem>
          </SelectGroup></SelectContent>
        </Select>
      </Field>
      <Button variant="outline" disabled={query.isFetching} onClick={() => query.refetch()}><RefreshCw data-icon="inline-start" />刷新</Button>
    </FieldGroup>
    {query.error && <ErrorInline error={query.error} onRetry={() => query.refetch()} />}
    {query.isLoading ? <LoadingStateWithBorder /> : query.data?.results.length === 0 ?
      <EmptyStateWithBorder title="暂无作者" description="镜像插画或小说后，会在对应栏目中显示作者。" /> :
      <div className="grid gap-4 md:grid-cols-2 xl:grid-cols-3">
        {query.data?.results.map(artist => <Card key={artist.artist_id}>
          <CardHeader>
            <div className="flex items-center gap-3">
              <Avatar><AvatarImage src={artist.avatar_url || undefined} alt={artist.name} /><AvatarFallback>{artist.name.slice(0, 1) || "作"}</AvatarFallback></Avatar>
              <div className="flex min-w-0 flex-col gap-1">
                <CardTitle><Link href={`/pixez/subscriptions/artist?artist_id=${artist.artist_id}&type=${target}`} className="hover:underline">{artist.name || `作者 ${artist.artist_id}`}</Link></CardTitle>
                <CardDescription>ID {artist.artist_id}</CardDescription>
              </div>
            </div>
          </CardHeader>
          <CardContent className="flex flex-col gap-2">
            <Badge variant={artist.subscription.enabled ? "secondary" : "outline"}>{artist.subscription.enabled ? "已订阅" : "未订阅"}</Badge>
            <p className="text-sm">已发现 {artist.work_count} 部 · 已完整备份 {artist.backup_count} 部</p>
            <p className="text-xs text-muted-foreground">最近成功扫描：{artist.subscription.last_success_at ? new Date(artist.subscription.last_success_at).toLocaleString() : "尚未扫描"}</p>
            {artist.subscription.last_error && <p className="text-sm text-destructive">{artist.subscription.last_error}</p>}
          </CardContent>
          <CardFooter><SubscriptionControls id={artist.artist_id} target={target} subscription={artist.subscription} /></CardFooter>
        </Card>)}
      </div>}
    <ArtistPagination page={page} total={query.data?.total ?? 0} onChange={setPage} />
  </div>
}
