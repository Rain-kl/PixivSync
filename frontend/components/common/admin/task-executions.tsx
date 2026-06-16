"use client"

import {useCallback, useEffect, useState} from "react"
import {toast} from "sonner"
import {format} from "date-fns"
import {Activity, ChevronLeft, ChevronRight, RefreshCw, RotateCcw} from "lucide-react"

import {AdminService, TaskExecution, TaskExecutionStatus, TaskMeta} from "@/lib/services"
import {ErrorInline} from "@/components/layout/error"
import {LoadingStateWithBorder} from "@/components/layout/loading"
import {EmptyStateWithBorder} from "@/components/layout/empty"
import {Badge} from "@/components/ui/badge"
import {Button} from "@/components/ui/button"
import {Label} from "@/components/ui/label"
import {Select, SelectContent, SelectItem, SelectTrigger, SelectValue} from "@/components/ui/select"
import {Sheet, SheetContent, SheetDescription, SheetFooter, SheetHeader, SheetTitle} from "@/components/ui/sheet"
import {Spinner} from "@/components/ui/spinner"
import {Table, TableBody, TableCell, TableHead, TableHeader, TableRow} from "@/components/ui/table"

const STATUS_LABELS: Record<TaskExecutionStatus, string> = {
  pending: "等待中",
  running: "执行中",
  succeeded: "成功",
  failed: "失败",
}

const TRIGGER_LABELS: Record<string, string> = {
  system: "系统",
  manual: "手动",
  retry: "重试",
  schedule: "定时",
}

function formatDateTime(value?: string | null) {
  if (!value) return "-"
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return value
  return format(date, "yyyy-MM-dd HH:mm:ss")
}

function formatDuration(duration: number) {
  if (!duration) return "-"
  if (duration < 1000) return `${ duration }ms`
  return `${ (duration / 1000).toFixed(2) }s`
}

function statusVariant(status: TaskExecutionStatus) {
  if (status === "failed") return "destructive"
  if (status === "succeeded") return "secondary"
  return "outline"
}

export function TaskExecutionsManager() {
  const [taskTypes, setTaskTypes] = useState<TaskMeta[]>([])
  const [executionsLoading, setExecutionsLoading] = useState(false)
  const [executionsError, setExecutionsError] = useState<Error | null>(null)
  const [executions, setExecutions] = useState<TaskExecution[]>([])
  const [executionsTotal, setExecutionsTotal] = useState(0)
  const [executionsPage, setExecutionsPage] = useState(1)
  const [executionStatus, setExecutionStatus] = useState<TaskExecutionStatus | "all">("all")
  const [executionTaskType, setExecutionTaskType] = useState<string>("all")
  const [selectedExecution, setSelectedExecution] = useState<TaskExecution | null>(null)
  const [detailOpen, setDetailOpen] = useState(false)
  const [detailLoading, setDetailLoading] = useState(false)
  const [retrying, setRetrying] = useState(false)

  const fetchTaskTypes = useCallback(async () => {
    try {
      const data = await AdminService.getTaskTypes()
      setTaskTypes(data)
    } catch {
      setTaskTypes([])
    }
  }, [])

  const fetchTaskExecutions = useCallback(async (overrides?: {
    page?: number
    status?: TaskExecutionStatus | "all"
    taskType?: string
  }) => {
    try {
      setExecutionsLoading(true)
      setExecutionsError(null)
      const page = overrides?.page ?? executionsPage
      const status = overrides?.status ?? executionStatus
      const taskType = overrides?.taskType ?? executionTaskType
      const data = await AdminService.listTaskExecutions({
        page,
        page_size: 10,
        status: status === "all" ? undefined : status,
        task_type: taskType === "all" ? undefined : taskType,
      })
      setExecutions(data.items || [])
      setExecutionsTotal(data.total || 0)
    } catch (err) {
      setExecutionsError(err instanceof Error ? err : new Error("加载任务执行记录失败"))
    } finally {
      setExecutionsLoading(false)
    }
  }, [executionStatus, executionTaskType, executionsPage])

  useEffect(() => {
    fetchTaskTypes()
  }, [fetchTaskTypes])

  useEffect(() => {
    fetchTaskExecutions()
  }, [fetchTaskExecutions])

  useEffect(() => {
    setExecutionsPage(1)
  }, [executionStatus, executionTaskType])

  const openExecutionDetail = async (execution: TaskExecution) => {
    setSelectedExecution(execution)
    setDetailOpen(true)
    try {
      setDetailLoading(true)
      const detail = await AdminService.getTaskExecution(execution.id)
      setSelectedExecution(detail)
    } catch (err) {
      toast.error("任务详情加载失败", {
        description: err instanceof Error ? err.message : "未知错误",
      })
    } finally {
      setDetailLoading(false)
    }
  }

  const handleRetryExecution = async () => {
    if (!selectedExecution) return
    try {
      setRetrying(true)
      const taskID = await AdminService.retryTaskExecution(selectedExecution.id)
      toast.success("任务已重新下发", {
        description: `新任务 ID：${ taskID }`,
      })
      await fetchTaskExecutions()
    } catch (err) {
      toast.error("任务重试失败", {
        description: err instanceof Error ? err.message : "未知错误",
      })
    } finally {
      setRetrying(false)
    }
  }

  const totalPages = Math.max(1, Math.ceil(executionsTotal / 10))

  return (
    <div className="space-y-6">
      <br/>
      <div className="flex flex-col gap-3 border-b border-border pb-3 sm:flex-row sm:items-center sm:justify-between">
        <div className="flex flex-wrap items-center gap-2">
          <Select value={executionStatus} onValueChange={(value) => setExecutionStatus(value as TaskExecutionStatus | "all")}>
            <SelectTrigger size="sm" className="w-[120px]">
              <SelectValue placeholder="状态" />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="all">全部状态</SelectItem>
              <SelectItem value="pending">等待中</SelectItem>
              <SelectItem value="running">执行中</SelectItem>
              <SelectItem value="succeeded">成功</SelectItem>
              <SelectItem value="failed">失败</SelectItem>
            </SelectContent>
          </Select>
          <Select value={executionTaskType} onValueChange={setExecutionTaskType}>
            <SelectTrigger size="sm" className="w-[180px]">
              <SelectValue placeholder="任务类型" />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="all">全部任务</SelectItem>
              {taskTypes.map((task) => (
                <SelectItem key={task.type} value={task.type}>{task.name || task.type}</SelectItem>
              ))}
            </SelectContent>
          </Select>
          <Button variant="outline" size="sm" onClick={() => fetchTaskExecutions()} disabled={executionsLoading}>
            {executionsLoading ? <Spinner className="size-4" /> : <RefreshCw className="size-4" />}
            刷新
          </Button>
        </div>
      </div>

      {executionsError ? (
        <div className="p-8 border border-dashed rounded-lg bg-card">
          <ErrorInline error={executionsError} onRetry={fetchTaskExecutions} className="justify-center" />
        </div>
      ) : executionsLoading && executions.length === 0 ? (
        <LoadingStateWithBorder icon={Activity} description="加载任务执行记录中..." />
      ) : executions.length === 0 ? (
        <EmptyStateWithBorder icon={Activity} description="暂无任务执行记录" />
      ) : (
        <div className="rounded-lg border bg-card">
          <Table className="min-w-[900px]">
            <TableHeader>
              <TableRow className="hover:bg-transparent">
                <TableHead className="w-[180px]">任务</TableHead>
                <TableHead className="w-[100px]">状态</TableHead>
                <TableHead className="w-[110px]">触发</TableHead>
                <TableHead className="w-[120px]">重试</TableHead>
                <TableHead className="w-[120px]">耗时</TableHead>
                <TableHead className="min-w-[220px]">结果/错误</TableHead>
                <TableHead className="w-[170px]">创建时间</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {executions.map((execution) => (
                <TableRow
                  key={execution.id}
                  className="cursor-pointer"
                  onClick={() => openExecutionDetail(execution)}
                >
                  <TableCell>
                    <div className="flex flex-col gap-1">
                      <span className="text-sm font-medium">{execution.task_name || execution.task_type}</span>
                      <span className="font-mono text-[11px] text-muted-foreground">{execution.task_id}</span>
                    </div>
                  </TableCell>
                  <TableCell>
                    <Badge variant={statusVariant(execution.status)}>
                      {STATUS_LABELS[execution.status] || execution.status}
                    </Badge>
                  </TableCell>
                  <TableCell>
                    <Badge variant="outline">{TRIGGER_LABELS[execution.triggered_by] || execution.triggered_by}</Badge>
                  </TableCell>
                  <TableCell className="font-mono text-xs text-muted-foreground">
                    {execution.retry_count}/{execution.max_retry}
                  </TableCell>
                  <TableCell className="font-mono text-xs text-muted-foreground">
                    {formatDuration(execution.duration)}
                  </TableCell>
                  <TableCell className="max-w-[320px] truncate text-xs text-muted-foreground">
                    {execution.error_message || execution.result || "-"}
                  </TableCell>
                  <TableCell className="font-mono text-[11px] text-muted-foreground">
                    {formatDateTime(execution.created_at)}
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        </div>
      )}

      <div className="flex items-center justify-between">
        <div className="text-xs text-muted-foreground">
          共 {executionsTotal} 条，当前第 {executionsPage}/{totalPages} 页
        </div>
        <div className="flex items-center gap-2">
          <Button
            variant="outline"
            size="sm"
            onClick={() => setExecutionsPage((page) => Math.max(1, page - 1))}
            disabled={executionsPage <= 1 || executionsLoading}
          >
            <ChevronLeft className="size-4" />
            上一页
          </Button>
          <Button
            variant="outline"
            size="sm"
            onClick={() => setExecutionsPage((page) => Math.min(totalPages, page + 1))}
            disabled={executionsPage >= totalPages || executionsLoading}
          >
            下一页
            <ChevronRight className="size-4" />
          </Button>
        </div>
      </div>

      <Sheet open={detailOpen} onOpenChange={setDetailOpen}>
        <SheetContent className="w-full p-0 sm:max-w-[640px]">
          <SheetHeader className="border-b">
            <SheetTitle>任务执行详情</SheetTitle>
            <SheetDescription>
              {selectedExecution?.task_name || selectedExecution?.task_type || "任务记录"}
            </SheetDescription>
          </SheetHeader>

          <div className="flex-1 overflow-y-auto px-4 pb-4">
            {detailLoading && !selectedExecution ? (
              <LoadingStateWithBorder icon={Activity} description="加载任务详情中..." />
            ) : selectedExecution ? (
              <div className="space-y-5 py-4">
                <div className="grid grid-cols-2 gap-3">
                  <div className="rounded-lg border p-3">
                    <div className="text-xs text-muted-foreground">状态</div>
                    <div className="mt-2">
                      <Badge variant={statusVariant(selectedExecution.status)}>
                        {STATUS_LABELS[selectedExecution.status] || selectedExecution.status}
                      </Badge>
                    </div>
                  </div>
                  <div className="rounded-lg border p-3">
                    <div className="text-xs text-muted-foreground">触发来源</div>
                    <div className="mt-2 text-sm font-medium">
                      {TRIGGER_LABELS[selectedExecution.triggered_by] || selectedExecution.triggered_by}
                    </div>
                  </div>
                  <div className="rounded-lg border p-3">
                    <div className="text-xs text-muted-foreground">重试次数</div>
                    <div className="mt-2 font-mono text-sm">
                      {selectedExecution.retry_count}/{selectedExecution.max_retry}
                    </div>
                  </div>
                  <div className="rounded-lg border p-3">
                    <div className="text-xs text-muted-foreground">耗时</div>
                    <div className="mt-2 font-mono text-sm">
                      {formatDuration(selectedExecution.duration)}
                    </div>
                  </div>
                </div>

                <div className="grid gap-2">
                  <Label>任务标识</Label>
                  <div className="rounded-md border bg-muted/40 px-3 py-2 font-mono text-xs break-all">
                    {selectedExecution.task_id}
                  </div>
                </div>

                <div className="grid grid-cols-1 gap-3 sm:grid-cols-2">
                  <div className="grid gap-2">
                    <Label>创建时间</Label>
                    <div className="font-mono text-xs text-muted-foreground">{formatDateTime(selectedExecution.created_at)}</div>
                  </div>
                  <div className="grid gap-2">
                    <Label>开始时间</Label>
                    <div className="font-mono text-xs text-muted-foreground">{formatDateTime(selectedExecution.started_at)}</div>
                  </div>
                  <div className="grid gap-2">
                    <Label>结束时间</Label>
                    <div className="font-mono text-xs text-muted-foreground">{formatDateTime(selectedExecution.finished_at)}</div>
                  </div>
                  <div className="grid gap-2">
                    <Label>更新时间</Label>
                    <div className="font-mono text-xs text-muted-foreground">{formatDateTime(selectedExecution.updated_at)}</div>
                  </div>
                </div>

                <div className="grid gap-2">
                  <Label>执行结果</Label>
                  <div className="min-h-10 rounded-md border bg-muted/30 px-3 py-2 text-sm whitespace-pre-wrap break-all">
                    {selectedExecution.result || "-"}
                  </div>
                </div>

                {selectedExecution.error_message && (
                  <div className="grid gap-2">
                    <Label>错误信息</Label>
                    <div className="rounded-md border border-destructive/30 bg-destructive/5 px-3 py-2 text-sm text-destructive whitespace-pre-wrap break-all">
                      {selectedExecution.error_message}
                    </div>
                  </div>
                )}

                <div className="grid gap-2">
                  <Label>Payload</Label>
                  <pre className="max-h-40 overflow-auto rounded-md border bg-muted/40 p-3 text-xs leading-relaxed">
                    {selectedExecution.payload || "{}"}
                  </pre>
                </div>

                <div className="grid gap-2">
                  <Label>执行日志</Label>
                  <pre className="min-h-48 max-h-[420px] overflow-auto rounded-md border bg-muted/40 p-3 text-xs leading-relaxed whitespace-pre-wrap">
                    {selectedExecution.log || "暂无日志"}
                  </pre>
                </div>
              </div>
            ) : (
              <EmptyStateWithBorder icon={Activity} description="未选择任务记录" />
            )}
          </div>

          <SheetFooter className="border-t">
            <Button variant="outline" onClick={() => selectedExecution && openExecutionDetail(selectedExecution)} disabled={!selectedExecution || detailLoading}>
              {detailLoading ? <Spinner className="size-4" /> : <RefreshCw className="size-4" />}
              刷新详情
            </Button>
            {selectedExecution && selectedExecution.status === "failed" && selectedExecution.retryable && (
              <Button onClick={handleRetryExecution} disabled={retrying}>
                {retrying ? <Spinner className="size-4" /> : <RotateCcw className="size-4" />}
                重试任务
              </Button>
            )}
          </SheetFooter>
        </SheetContent>
      </Sheet>
    </div>
  )
}
