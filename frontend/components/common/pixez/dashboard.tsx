"use client"

import {BarChart3, RefreshCw} from "lucide-react"

import {Button} from "@/components/ui/button"
import {ErrorInline} from "@/components/layout/error"
import {Spinner} from "@/components/ui/spinner"

import {usePixEzStats} from "./api/usePixEzStats"
import {formatPixEzDateTime} from "./pixez-format"
import {RecentExportRunsTable} from "./RecentExportRunsTable"
import {StatsCards} from "./StatsCards"

export function PixEzDashboard() {
  const dashboardQuery = usePixEzStats()

  return (
    <div className="flex w-full flex-col gap-5 py-6">
      <div className="flex items-center gap-3 border-b border-border pb-3">
        <div className="flex size-10 items-center justify-center rounded-md border bg-card">
          <BarChart3 className="text-muted-foreground" />
        </div>
        <div className="flex flex-col gap-1">
          <h1 className="text-2xl font-semibold tracking-tight">同步看板</h1>
          <p className="text-xs text-muted-foreground">镜像统计、队列状态与最近同步批次</p>
        </div>
      </div>

      <div className="flex flex-col gap-4">
        <div className="flex flex-col gap-2 sm:flex-row sm:items-center sm:justify-between">
          <div className="text-xs text-muted-foreground">
            更新于 {formatPixEzDateTime(dashboardQuery.data?.updated_at)}
          </div>
          <Button variant="outline" size="sm" onClick={() => dashboardQuery.refetch()} disabled={dashboardQuery.isFetching}>
            {dashboardQuery.isFetching ? <Spinner /> : <RefreshCw />}
            刷新
          </Button>
        </div>

        {dashboardQuery.error && (
          <div className="rounded-md border p-3">
            <ErrorInline error={dashboardQuery.error} onRetry={() => dashboardQuery.refetch()} />
          </div>
        )}

        <StatsCards dashboard={dashboardQuery.data} loading={dashboardQuery.isLoading} />
        <RecentExportRunsTable runs={dashboardQuery.data?.recent_runs ?? []} />
      </div>
    </div>
  )
}
