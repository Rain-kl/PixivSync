"use client"

import {useState} from "react"
import {useQueryClient} from "@tanstack/react-query"
import {RefreshCw, UsersRound, Zap} from "lucide-react"
import {toast} from "sonner"

import {Button} from "@/components/ui/button"
import {Spinner} from "@/components/ui/spinner"
import {EmptyStateWithBorder} from "@/components/layout/empty"
import {ErrorInline} from "@/components/layout/error"
import {LoadingStateWithBorder} from "@/components/layout/loading"
import {AdminService, PixezService} from "@/lib/services"
import type {PixezAccount} from "@/lib/services"

import {AccountCard} from "./AccountCard"
import {usePixEzAccounts} from "./api/usePixEzAccounts"


export function PixEzAccounts() {
  const queryClient = useQueryClient()
  const accountsQuery = usePixEzAccounts()
  const [syncingID, setSyncingID] = useState<string | null>(null)
  const [refreshingID, setRefreshingID] = useState<string | null>(null)
  const [deletingID, setDeletingID] = useState<string | null>(null)
  const accounts = accountsQuery.data ?? []

  const invalidatePixEz = async () => {
    await Promise.all([
      queryClient.invalidateQueries({queryKey: ["pixez"]}),
      queryClient.invalidateQueries({queryKey: ["admin", "task-executions"]}),
    ])
  }

  const handleSync = async (account?: PixezAccount) => {
    const id = account?.pixiv_user_id ?? "all"
    try {
      setSyncingID(id)
      const payload = account ? JSON.stringify({pixiv_user_id: account.pixiv_user_id}) : ""
      const [illustTaskID, novelTaskID] = await Promise.all([
        AdminService.dispatchTask({
          task_type: "pixez_export_bookmark_illusts",
          payload,
        }),
        AdminService.dispatchTask({
          task_type: "pixez_export_bookmark_novels",
          payload,
        }),
      ])
      toast.success("收藏同步任务已入队", {
        description: `${illustTaskID} / ${novelTaskID}`,
      })
      await invalidatePixEz()
    } catch (error) {
      toast.error("同步收藏失败", {
        description: error instanceof Error ? error.message : "未知错误",
      })
    } finally {
      setSyncingID(null)
    }
  }

  const handleRefresh = async (account: PixezAccount) => {
    try {
      setRefreshingID(account.pixiv_user_id)
      await PixezService.refreshAccountToken(account.pixiv_user_id)
      toast.success("Pixiv 凭证已刷新")
      await queryClient.invalidateQueries({queryKey: ["pixez", "accounts"]})
    } catch (error) {
      toast.error("刷新凭证失败", {
        description: error instanceof Error ? error.message : "未知错误",
      })
    } finally {
      setRefreshingID(null)
    }
  }

  const handleDelete = async (account: PixezAccount) => {
    try {
      setDeletingID(account.pixiv_user_id)
      await PixezService.deleteAccount(account.pixiv_user_id)
      toast.success("Pixiv 账号已断开")
      await invalidatePixEz()
    } catch (error) {
      toast.error("断开账号失败", {
        description: error instanceof Error ? error.message : "未知错误",
      })
    } finally {
      setDeletingID(null)
    }
  }

  return (
    <div className="flex w-full flex-col gap-5 py-6">
      <div className="flex items-center gap-3 border-b border-border pb-3">
        <div className="flex size-10 items-center justify-center rounded-md border bg-card">
          <UsersRound className="text-muted-foreground" />
        </div>
        <div className="flex flex-col gap-1">
          <h1 className="text-2xl font-semibold tracking-tight">账号管理</h1>
          <p className="text-xs text-muted-foreground">Pixiv 账号授权与数据导入同步</p>
        </div>
      </div>

      <div className="flex flex-col gap-4">
        <div className="flex flex-col gap-2 sm:flex-row sm:items-center sm:justify-between">
          <div className="text-sm font-medium">账号同步管理</div>
          <div className="flex flex-wrap gap-2">
            <Button variant="outline" size="sm" onClick={() => accountsQuery.refetch()} disabled={accountsQuery.isFetching}>
              {accountsQuery.isFetching ? <Spinner /> : <RefreshCw />}
              刷新
            </Button>
            <Button size="sm" onClick={() => handleSync()} disabled={syncingID !== null || accounts.length === 0}>
              {syncingID === "all" ? <Spinner /> : <Zap />}
              同步全部
            </Button>
          </div>
        </div>

        {accountsQuery.error ? (
          <div className="rounded-md border p-3">
            <ErrorInline error={accountsQuery.error} onRetry={() => accountsQuery.refetch()} />
          </div>
        ) : accountsQuery.isLoading ? (
          <LoadingStateWithBorder description="加载 Pixiv 账号中..." />
        ) : accounts.length === 0 ? (
          <EmptyStateWithBorder description="暂无 Pixiv 账号" />
        ) : (
          <div className="grid gap-3 lg:grid-cols-2 2xl:grid-cols-3">
            {accounts.map((account) => (
              <AccountCard
                key={account.pixiv_user_id}
                account={account}
                syncing={syncingID === account.pixiv_user_id}
                refreshing={refreshingID === account.pixiv_user_id}
                deleting={deletingID === account.pixiv_user_id}
                onSync={handleSync}
                onRefresh={handleRefresh}
                onDelete={handleDelete}
              />
            ))}
          </div>
        )}
      </div>
    </div>
  )
}
