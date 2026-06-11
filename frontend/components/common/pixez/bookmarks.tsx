"use client"

import {useMemo, useState} from "react"
import {RefreshCw, Search, Heart} from "lucide-react"

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

import {usePixEzBookmarks} from "./api/usePixEzBookmarks"
import {usePixEzAccounts} from "./api/usePixEzAccounts"
import {BookmarkGallery} from "./BookmarkGallery"

type MirrorStatusFilter = PixezMirrorStatusText | "all"
type WorkStatusFilter = NonNullable<PixezBookmarkQuery["work_status"]>

const pageSize = 24

export function PixEzBookmarks() {
  const [target, setTarget] = useState<PixezMirrorTarget>("illust")
  const [page, setPage] = useState(1)
  const [query, setQuery] = useState("")
  const [mirrorStatus, setMirrorStatus] = useState<MirrorStatusFilter>("all")
  const [workStatus, setWorkStatus] = useState<WorkStatusFilter>("active")
  const [accountFilter, setAccountFilter] = useState<string>("all")

  // Fetch Pixiv accounts
  const accountsQuery = usePixEzAccounts()
  const accounts = accountsQuery.data ?? []

  const params = useMemo<PixezBookmarkQuery>(() => ({
    page,
    page_size: pageSize,
    q: query.trim(),
    mirror_status: mirrorStatus,
    work_status: workStatus,
    pixiv_user_id: accountFilter === "all" ? undefined : accountFilter,
  }), [mirrorStatus, page, query, workStatus, accountFilter])

  const bookmarksQuery = usePixEzBookmarks(target, params)
  const items = bookmarksQuery.data?.items ?? []

  const resetPage = () => setPage(1)

  return (
    <div className="flex w-full flex-col gap-5 py-6">
      <div className="flex items-center gap-3 border-b border-border pb-3">
        <div className="flex size-10 items-center justify-center rounded-md border bg-card">
          <Heart className="text-muted-foreground" />
        </div>
        <div className="flex flex-col gap-1">
          <h1 className="text-2xl font-semibold tracking-tight">收藏管理</h1>
          <p className="text-xs text-muted-foreground">管理并查看从 Pixiv 同步的收藏作品和镜像预览</p>
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

              <div className="grid gap-2 sm:grid-cols-[minmax(200px,1fr)_150px_150px_150px_auto] xl:min-w-[820px]">
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

                {/* Account Selector */}
                <Select
                  value={accountFilter}
                  onValueChange={(value) => {
                    setAccountFilter(value)
                    resetPage()
                  }}
                  disabled={accountsQuery.isLoading}
                >
                  <SelectTrigger className="w-full">
                    <SelectValue placeholder="全部账号" />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectGroup>
                      <SelectItem value="all">全部账号</SelectItem>
                      {accounts.map((acc) => (
                        <SelectItem key={acc.pixiv_user_id} value={acc.pixiv_user_id}>
                          {acc.name || acc.account}
                        </SelectItem>
                      ))}
                    </SelectGroup>
                  </SelectContent>
                </Select>

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
                      <SelectItem value="all">全部镜像</SelectItem>
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

                <Button variant="outline" onClick={() => bookmarksQuery.refetch()} disabled={bookmarksQuery.isFetching}>
                  {bookmarksQuery.isFetching ? <Spinner /> : <RefreshCw />}
                  刷新
                </Button>
              </div>
            </div>

            {bookmarksQuery.error && (
              <div className="rounded-md border p-3">
                <ErrorInline error={bookmarksQuery.error} onRetry={() => bookmarksQuery.refetch()} />
              </div>
            )}
          </CardContent>
        </Card>

        <BookmarkGallery
          target={target}
          items={items}
          total={bookmarksQuery.data?.total ?? 0}
          page={page}
          pageSize={pageSize}
          loading={bookmarksQuery.isLoading || bookmarksQuery.isFetching}
          onPageChange={setPage}
        />
      </div>
    </div>
  )
}
