"use client"

import {useMemo, useState} from "react"
import {RefreshCw, Search, GalleryVerticalEnd} from "lucide-react"

import {Button} from "@/components/ui/button"
import {Card, CardContent} from "@/components/ui/card"
import {Input} from "@/components/ui/input"
import {
  Select,
  SelectContent,
  SelectGroup,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select"
import {Spinner} from "@/components/ui/spinner"
import {Tabs, TabsList, TabsTrigger} from "@/components/ui/tabs"
import {ErrorInline} from "@/components/layout/error"
import type {PixezBookmarkQuery, PixezMirrorStatusText, PixezMirrorTarget} from "@/lib/services"

import {usePixEzMirrors} from "./api/usePixEzMirrors"
import {MirrorGallery} from "./MirrorGallery"

type MirrorStatusFilter = PixezMirrorStatusText | "all"
type WorkStatusFilter = NonNullable<PixezBookmarkQuery["work_status"]>

const pageSize = 24

export function PixEzMirrors() {
  const [target, setTarget] = useState<PixezMirrorTarget>("illust")
  const [page, setPage] = useState(1)
  const [query, setQuery] = useState("")
  const [mirrorStatus, setMirrorStatus] = useState<MirrorStatusFilter>("all")
  const [workStatus, setWorkStatus] = useState<WorkStatusFilter>("active")

  const params = useMemo<PixezBookmarkQuery>(() => ({
    page,
    page_size: pageSize,
    q: query.trim(),
    mirror_status: mirrorStatus,
    work_status: workStatus,
  }), [mirrorStatus, page, query, workStatus])

  const mirrorsQuery = usePixEzMirrors(target, params)
  const items = mirrorsQuery.data?.items ?? []

  const resetPage = () => setPage(1)

  return (
    <div className="flex w-full flex-col gap-5 py-6">
      <div className="flex items-center gap-3 border-b border-border pb-3">
        <div className="flex size-10 items-center justify-center rounded-md border bg-card">
          <GalleryVerticalEnd className="text-muted-foreground" />
        </div>
        <div className="flex flex-col gap-1">
          <h1 className="text-2xl font-semibold tracking-tight">镜像数据</h1>
          <p className="text-xs text-muted-foreground">已镜像的插画与小说元数据及资源状态</p>
        </div>
      </div>

      <div className="flex flex-col gap-4">
        <Card className="rounded-lg">
          <CardContent className="flex flex-col gap-3 py-4">
            <div className="flex flex-col gap-3 xl:flex-row xl:items-center xl:justify-between">
              <Tabs
                value={target}
                onValueChange={(value) => {
                  setTarget(value as PixezMirrorTarget)
                  setPage(1)
                }}
                className="w-full xl:w-auto"
              >
                <TabsList className="grid w-full grid-cols-2 xl:w-[220px]">
                  <TabsTrigger value="illust">插画管理</TabsTrigger>
                  <TabsTrigger value="novel">小说管理</TabsTrigger>
                </TabsList>
              </Tabs>

              <div className="grid gap-2 sm:grid-cols-[minmax(220px,1fr)_160px_160px_auto] xl:min-w-[760px]">
                <div className="relative">
                  <Search className="pointer-events-none absolute left-2.5 top-1/2 -translate-y-1/2 text-muted-foreground" />
                  <Input
                    value={query}
                    onChange={(event) => {
                      setQuery(event.target.value)
                      resetPage()
                    }}
                    className="pl-8"
                    placeholder="ID、标题、作者"
                  />
                </div>

                <Select
                  value={mirrorStatus}
                  onValueChange={(value) => {
                    setMirrorStatus(value as MirrorStatusFilter)
                    resetPage()
                  }}
                >
                  <SelectTrigger className="w-full">
                    <SelectValue placeholder="镜像状态" />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectGroup>
                      <SelectItem value="all">全部状态</SelectItem>
                      <SelectItem value="success">成功</SelectItem>
                      <SelectItem value="processing">下载中</SelectItem>
                      <SelectItem value="failed">失败</SelectItem>
                      <SelectItem value="none">未入队</SelectItem>
                    </SelectGroup>
                  </SelectContent>
                </Select>

                <Select
                  value={workStatus}
                  onValueChange={(value) => {
                    setWorkStatus(value as WorkStatusFilter)
                    resetPage()
                  }}
                >
                  <SelectTrigger className="w-full">
                    <SelectValue placeholder="作品状态" />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectGroup>
                      <SelectItem value="active">当前收藏</SelectItem>
                      <SelectItem value="visible">可见</SelectItem>
                      <SelectItem value="muted">已屏蔽</SelectItem>
                      <SelectItem value="unavailable">不可见</SelectItem>
                      <SelectItem value="removed">已移除</SelectItem>
                      <SelectItem value="all">全部作品</SelectItem>
                    </SelectGroup>
                  </SelectContent>
                </Select>

                <Button variant="outline" onClick={() => mirrorsQuery.refetch()} disabled={mirrorsQuery.isFetching}>
                  {mirrorsQuery.isFetching ? <Spinner /> : <RefreshCw />}
                  刷新
                </Button>
              </div>
            </div>

            {mirrorsQuery.error && (
              <div className="rounded-md border p-3">
                <ErrorInline error={mirrorsQuery.error} onRetry={() => mirrorsQuery.refetch()} />
              </div>
            )}
          </CardContent>
        </Card>

        <MirrorGallery
          target={target}
          items={items}
          total={mirrorsQuery.data?.total ?? 0}
          page={page}
          pageSize={pageSize}
          loading={mirrorsQuery.isLoading || mirrorsQuery.isFetching}
          onPageChange={setPage}
        />
      </div>
    </div>
  )
}
