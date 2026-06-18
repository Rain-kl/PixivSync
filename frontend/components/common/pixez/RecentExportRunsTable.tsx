"use client"

import Link from "next/link"
import {useState} from "react"
import {useQueryClient} from "@tanstack/react-query"
import {ExternalLink, RotateCcw} from "lucide-react"
import {toast} from "sonner"

import {Badge} from "@/components/ui/badge"
import {Button} from "@/components/ui/button"
import {Card, CardContent, CardDescription, CardHeader, CardTitle} from "@/components/ui/card"
import {Spinner} from "@/components/ui/spinner"
import {Table, TableBody, TableCell, TableHead, TableHeader, TableRow} from "@/components/ui/table"
import {EmptyStateWithBorder} from "@/components/layout/empty"
import type {PixezExportRun, PixezRunStatus} from "@/lib/services"
import {AdminTaskService} from "@/lib/services"
import {useUser} from "@/contexts/user-context"

import {formatPixEzDateTime, formatPixEzDuration, pixezRunStatusLabel, pixezTargetLabel} from "./pixez-format"

function runStatusVariant(status: PixezRunStatus) {
  if (status === "failed") return "destructive"
  if (status === "success") return "secondary"
  return "outline"
}

function taskTypeForRun(run: PixezExportRun) {
  return run.target_type === "illust"
    ? "pixez_export_bookmark_illusts"
    : "pixez_export_bookmark_novels"
}

export function RecentExportRunsTable({runs}: {runs: PixezExportRun[]}) {
  const queryClient = useQueryClient()
  const {user} = useUser()
  const [retryingID, setRetryingID] = useState<string | null>(null)

  const handleRetry = async (run: PixezExportRun) => {
    try {
      setRetryingID(run.id)
      const taskID = await AdminTaskService.dispatchTask({
        task_type: taskTypeForRun(run),
        payload: JSON.stringify({pixiv_user_id: run.pixiv_user_id}),
      })
      toast.success("同步任务已重新入队", {
        description: `任务 ID：${taskID}`,
      })
      await Promise.all([
        queryClient.invalidateQueries({queryKey: ["pixez", "dashboard"]}),
        queryClient.invalidateQueries({queryKey: ["pixez", "export-runs"]}),
      ])
    } catch (error) {
      toast.error("强制重试失败", {
        description: error instanceof Error ? error.message : "未知错误",
      })
    } finally {
      setRetryingID(null)
    }
  }

  return (
    <Card className="rounded-lg">
      <CardHeader>
        <CardTitle>最近同步批次</CardTitle>
        <CardDescription>bookmark_export_runs</CardDescription>
      </CardHeader>
      <CardContent>
        {runs.length === 0 ? (
          <EmptyStateWithBorder description="暂无同步批次记录" />
        ) : (
          <div className="rounded-md border">
            <Table className="min-w-[860px]">
              <TableHeader>
                <TableRow className="hover:bg-transparent">
                  <TableHead className="w-[210px]">记录 ID</TableHead>
                  <TableHead className="w-[90px]">类型</TableHead>
                  <TableHead className="w-[120px]">账号</TableHead>
                  <TableHead className="w-[90px]">状态</TableHead>
                  <TableHead className="w-[150px]">数据增量</TableHead>
                  <TableHead className="w-[150px]">开始时间</TableHead>
                  <TableHead className="w-[90px]">耗时</TableHead>
                  <TableHead className="w-[150px] text-right">操作</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {runs.map((run) => (
                  <TableRow key={run.id}>
                    <TableCell className="font-mono text-xs">{run.id}</TableCell>
                    <TableCell>{pixezTargetLabel(run.target_type)}</TableCell>
                    <TableCell className="font-mono text-xs">{run.pixiv_user_id || "-"}</TableCell>
                    <TableCell>
                      <Badge variant={runStatusVariant(run.status)}>
                        {pixezRunStatusLabel(run.status)}
                      </Badge>
                    </TableCell>
                    <TableCell className="font-mono text-xs text-muted-foreground">
                      +{run.new_count} / ~{run.updated_count} / -{run.removed_count}
                    </TableCell>
                    <TableCell className="font-mono text-[11px] text-muted-foreground">
                      {formatPixEzDateTime(run.started_at)}
                    </TableCell>
                    <TableCell className="font-mono text-xs text-muted-foreground">
                      {formatPixEzDuration(run.duration_ms)}
                    </TableCell>
                    <TableCell>
                      <div className="flex justify-end gap-2">
                        {user?.is_admin && (
                          <Button variant="outline" size="sm" asChild>
                            <Link href={`/admin/tasks/executions?task_type=${taskTypeForRun(run)}`}>
                              <ExternalLink />
                              日志
                            </Link>
                          </Button>
                        )}
                        <Button
                          variant="outline"
                          size="sm"
                          onClick={() => handleRetry(run)}
                          disabled={retryingID === run.id}
                        >
                          {retryingID === run.id ? <Spinner /> : <RotateCcw />}
                          重试
                        </Button>
                      </div>
                    </TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          </div>
        )}
      </CardContent>
    </Card>
  )
}
