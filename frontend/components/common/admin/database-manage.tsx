"use client"

import * as React from "react"
import {useCallback, useEffect, useRef, useState} from "react"
import {useTheme} from "next-themes"
import CodeMirror from "@uiw/react-codemirror"
import {sql} from "@codemirror/lang-sql"
import {toast} from "sonner"
import {
  Activity,
  ArrowLeft,
  Cpu,
  Database,
  Download,
  FileText,
  HardDrive,
  Layers,
  Play,
  RefreshCw,
  Server,
  Terminal,
  Trash2,
} from "lucide-react"
import {Card, CardContent, CardDescription, CardHeader, CardTitle} from "@/components/ui/card"
import {Button} from "@/components/ui/button"
import {Skeleton} from "@/components/ui/skeleton"
import {Select, SelectContent, SelectItem, SelectTrigger, SelectValue} from "@/components/ui/select"
import {Table, TableBody, TableCell, TableHead, TableHeader, TableRow} from "@/components/ui/table"
import {Input} from "@/components/ui/input"
import {Switch} from "@/components/ui/switch"
import {Label} from "@/components/ui/label"
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog"
import type {CacheStatus, DBOverview, ExecuteSQLResponse, TableDataResponse} from "@/lib/services"
import services, {AdminService} from "@/lib/services"

/**
 * 格式化数字，每3位加逗号
 */
const formatNumber = (num: number | string) => {
  if (num === undefined || num === null) return "0"
  return num.toString().replace(/\B(?=(\d{3})+(?!\B))/g, ",")
}

/**
 * 格式化表格单元格内容
 */
const formatCellValue = (val: unknown): string => {
  if (val === null || val === undefined) return "-"
  if (typeof val === "boolean") return val ? "true" : "false"
  if (typeof val === "object") {
    try {
      return JSON.stringify(val)
    } catch {
      return "[Object]"
    }
  }
  return String(val)
}

export function DatabaseManager() {
  // 基础状态
  const [overview, setOverview] = useState<DBOverview | null>(null)
  const [tables, setTables] = useState<string[]>([])
  const [selectedTable, setSelectedTable] = useState<string>("")
  const [tableData, setTableData] = useState<TableDataResponse | null>(null)

  // 分页状态
  const [page, setPage] = useState<number>(1)
  const pageSize = 10

  // 加载状态
  const [loadingOverview, setLoadingOverview] = useState<boolean>(true)
  const [loadingTables, setLoadingTables] = useState<boolean>(true)
  const [loadingData, setLoadingData] = useState<boolean>(false)
  const [exporting, setExporting] = useState<boolean>(false)

  // SQL 控制台状态
  const [showConsole, setShowConsole] = useState<boolean>(false)
  const [sqlQuery, setSqlQuery] = useState<string>("")
  const [executingSQL, setExecutingSQL] = useState<boolean>(false)
  const [sqlResult, setSqlResult] = useState<ExecuteSQLResponse | null>(null)
  const [sqlError, setSqlError] = useState<string | null>(null)

  // 缓存管理状态
  const [cacheStatus, setCacheStatus] = useState<CacheStatus | null>(null)
  const [loadingCache, setLoadingCache] = useState<boolean>(false)
  const [savingConfig, setSavingConfig] = useState<boolean>(false)
  const [clearingCache, setClearingCache] = useState<boolean>(false)
  const [maxSizeMB, setMaxSizeMB] = useState<string>("100")
  const [ttlMinutes, setTtlMinutes] = useState<string>("60")
  const [lruEnabled, setLruEnabled] = useState<boolean>(true)
  const [showClearConfirm, setShowClearConfirm] = useState<boolean>(false)

  // 拖拽与主题状态
  const containerRef = useRef<HTMLDivElement>(null)
  const [editorHeight, setEditorHeight] = useState<number>(240)
  const { resolvedTheme } = useTheme()
  const cmTheme = resolvedTheme === "dark" ? "dark" : "light"

  const handleMouseDown = (e: React.MouseEvent) => {
    e.preventDefault()
    const startY = e.clientY
    const startHeight = editorHeight

    const handleMouseMove = (moveEvent: MouseEvent) => {
      if (!containerRef.current) return
      const deltaY = moveEvent.clientY - startY
      const containerHeight = containerRef.current.getBoundingClientRect().height
      const newHeight = startHeight + deltaY

      // Limit editor height between 80px and containerHeight - 80px
      if (newHeight > 80 && newHeight < containerHeight - 80) {
        setEditorHeight(newHeight)
      }
    }

    const handleMouseUp = () => {
      document.removeEventListener("mousemove", handleMouseMove)
      document.removeEventListener("mouseup", handleMouseUp)
    }

    document.addEventListener("mousemove", handleMouseMove)
    document.addEventListener("mouseup", handleMouseUp)
  }

  // 1. 获取运行概览
  const fetchOverview = useCallback(async (isSilent = false) => {
    if (!isSilent) setLoadingOverview(true)
    try {
      const data = await services.dbManage.getOverview()
      setOverview(data)
    } catch (err) {
      toast.error("获取数据库概览失败", {
        description: err instanceof Error ? err.message : "未知错误",
      })
    } finally {
      setLoadingOverview(false)
    }
  }, [])

  // 2. 获取表列表
  const fetchTables = useCallback(async () => {
    setLoadingTables(true)
    try {
      const data = await services.dbManage.listTables()
      setTables(data)
      // 默认选择第一张表
      if (data.length > 0 && !selectedTable) {
        setSelectedTable(data[0])
      }
    } catch (err) {
      toast.error("获取数据库数据表列表失败", {
        description: err instanceof Error ? err.message : "未知错误",
      })
    } finally {
      setLoadingTables(false)
    }
  }, [selectedTable])

  // 3. 获取具体表的数据
  const fetchTableData = useCallback(async (tableName: string, targetPage: number, size: number) => {
    if (!tableName) return
    setLoadingData(true)
    try {
      const data = await services.dbManage.getTableData({
        table: tableName,
        page: targetPage,
        pageSize: size,
      })
      setTableData(data)
    } catch (err) {
      toast.error(`获取数据表 ${tableName} 数据失败`, {
        description: err instanceof Error ? err.message : "未知错误",
      })
    } finally {
      setLoadingData(false)
    }
  }, [])

  // 4. 导出数据库
  const handleExport = async () => {
    setExporting(true)
    try {
      const { blob, filename } = await AdminService.exportDatabase()
      const url = URL.createObjectURL(blob)
      const a = document.createElement("a")
      a.href = url
      a.download = filename
      document.body.appendChild(a)
      a.click()
      a.remove()
      URL.revokeObjectURL(url)
      toast.success("数据库导出成功", { description: `已下载归档文件: ${filename}` })
    } catch (err) {
      toast.error("数据库导出失败", {
        description: err instanceof Error ? err.message : "导出异常",
      })
    } finally {
      setExporting(false)
    }
  }

  // 5. 执行 SQL 查询
  const handleExecuteSQL = async () => {
    if (!sqlQuery.trim()) return
    setExecutingSQL(true)
    setSqlResult(null)
    setSqlError(null)
    try {
      const result = await services.dbManage.executeSQL(sqlQuery)
      setSqlResult(result)
      toast.success("SQL 执行成功")
    } catch (err) {
      setSqlError(err instanceof Error ? err.message : "未知执行错误")
      toast.error("SQL 执行失败")
    } finally {
      setExecutingSQL(false)
    }
  }

  // 6. 格式化字节大小
  const formatBytes = (bytes: number) => {
    if (bytes === 0) return "0 Bytes"
    const k = 1024
    const sizes = ["Bytes", "KB", "MB", "GB", "TB"]
    const i = Math.floor(Math.log(bytes) / Math.log(k))
    return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + " " + sizes[i]
  }

  // 7. 获取磁盘缓存状态
  const fetchCacheStatus = useCallback(async (isSilent = false) => {
    if (!isSilent) setLoadingCache(true)
    try {
      const data = await AdminService.getCacheStatus()
      setCacheStatus(data)
      setMaxSizeMB(data.max_size_mb.toString())
      setTtlMinutes(data.ttl_minutes.toString())
      setLruEnabled(data.lru_enabled)
    } catch (err) {
      toast.error("获取磁盘缓存状态失败", {
        description: err instanceof Error ? err.message : "未知错误",
      })
    } finally {
      setLoadingCache(false)
    }
  }, [])

  // 8. 保存磁盘缓存配置
  const handleSaveConfig = async (e: React.FormEvent) => {
    e.preventDefault()
    const size = parseInt(maxSizeMB, 10)
    const ttl = parseInt(ttlMinutes, 10)
    if (isNaN(size) || size < 1) {
      toast.error("保存失败", { description: "最大容量限制必须是大于等于 1 的整数" })
      return
    }
    if (isNaN(ttl) || ttl < 0) {
      toast.error("保存失败", { description: "默认过期时间必须是大于等于 0 的整数" })
      return
    }
    setSavingConfig(true)
    try {
      await AdminService.updateCacheConfig({
        max_size_mb: size,
        ttl_minutes: ttl,
        lru_enabled: lruEnabled,
      })
      toast.success("保存成功", { description: "磁盘缓存策略已热更新" })
      await fetchCacheStatus(true)
    } catch (err) {
      toast.error("保存配置失败", {
        description: err instanceof Error ? err.message : "未知错误",
      })
    } finally {
      setSavingConfig(false)
    }
  }

  // 9. 清空磁盘缓存数据
  const handleClearCache = async () => {
    setClearingCache(true)
    try {
      await AdminService.clearCache()
      toast.success("清空成功", { description: "缓存数据已全部清除" })
      setShowClearConfirm(false)
      await fetchCacheStatus(true)
    } catch (err) {
      toast.error("清空缓存失败", {
        description: err instanceof Error ? err.message : "未知错误",
      })
    } finally {
      setClearingCache(false)
    }
  }

  // 页面初始化加载
  useEffect(() => {
    fetchOverview()
    fetchTables()
    fetchCacheStatus()
  }, [fetchOverview, fetchTables, fetchCacheStatus])

  // 选择数据表或页码变化时，拉取对应的数据
  useEffect(() => {
    if (selectedTable) {
      fetchTableData(selectedTable, page, pageSize)
    }
  }, [selectedTable, page, pageSize, fetchTableData])

  // 处理页码改变
  const handlePageChange = (newPage: number) => {
    setPage(newPage)
  }

  // 处理表切换
  const handleTableChange = (value: string) => {
    setSelectedTable(value)
    setPage(1) // 切换表重置为第一页
  }

  // Preset SQL 选择器
  const handlePresetSQLChange = (query: string) => {
    setSqlQuery(query)
  }

  // 渲染概览骨架
  const renderOverviewSkeleton = () => (
    <div className="grid grid-cols-2 md:grid-cols-3 lg:grid-cols-6 gap-4">
      {Array.from({ length: 6 }).map((_, i) => (
        <Card key={i} className="border-border/40 bg-card/50">
          <CardHeader className="pb-2">
            <Skeleton className="h-4 w-16" />
          </CardHeader>
          <CardContent>
            <Skeleton className="h-6 w-24" />
          </CardContent>
        </Card>
      ))}
    </div>
  )

  // SQL 查询控台渲染
  if (showConsole) {
    return (
      <div className="space-y-3 p-1 w-full">
        {/* 顶部控制与标题 */}
        <div className="flex items-center justify-between border-b border-border/30 pb-2">
          <div className="flex items-center gap-3">
            <Button
              variant="outline"
              size="icon"
              className="h-8 w-8"
              onClick={() => {
                setShowConsole(false)
                setSqlResult(null)
                setSqlError(null)
              }}
            >
              <ArrowLeft className="size-4" />
            </Button>
            <div>
              <h2 className="text-xl font-bold tracking-tight">SQL 查询终端</h2>
              <p className="text-xs text-muted-foreground">执行自定义 SQL 查询，支持数据读取与写入操作</p>
            </div>
          </div>
          <div className="text-xs text-muted-foreground font-mono">
            {overview?.type === "postgres" ? "PostgreSQL Connected" : "SQLite Connected"}
          </div>
        </div>

        {/* 类似于 VS Code 的单个整体编辑器+结果区域，固定高度，有分水岭拖拽调整大小 */}
        <div
          ref={containerRef}
          className="w-full border border-border/40 bg-card/60 backdrop-blur-md rounded-lg overflow-hidden flex flex-col shadow-sm h-[calc(100vh-140px)] min-h-[500px]"
        >
          {/* 顶部编辑工具栏 */}
          <div className="flex items-center justify-between px-4 py-2 border-b bg-muted/40 shrink-0 gap-4 flex-wrap">
            <div className="flex items-center gap-2">
              <Terminal className="size-4 text-primary" />
              <span className="text-xs font-semibold">SQL 编辑器</span>
            </div>

            <div className="flex items-center gap-3 flex-wrap">
              {/* 快速模板 */}
              <div className="flex items-center gap-1.5">
                <span className="text-[11px] text-muted-foreground">快速模板:</span>
                <Select onValueChange={handlePresetSQLChange}>
                  <SelectTrigger className="h-7 w-[180px] text-[11px] bg-background">
                    <SelectValue placeholder="选择预设 SQL" />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem value="SELECT * FROM users LIMIT 10;">查询用户 (SELECT users)</SelectItem>
                    <SelectItem value="SELECT * FROM uploads LIMIT 10;">查询文件 (SELECT uploads)</SelectItem>
                    <SelectItem value="SELECT * FROM task_executions ORDER BY created_at DESC LIMIT 10;">查询任务流水 (SELECT task_executions)</SelectItem>
                    <SelectItem value="SELECT sqlite_version();">SQLite 版本 (SQLite only)</SelectItem>
                    <SelectItem value="SELECT version();">PostgreSQL 版本 (Postgres only)</SelectItem>
                  </SelectContent>
                </Select>
              </div>

              {/* 控制按钮 */}
              <div className="flex items-center gap-1.5">
                <Button
                  variant="ghost"
                  size="icon"
                  className="h-7 w-7 text-muted-foreground hover:text-foreground"
                  onClick={() => setSqlQuery("")}
                  title="清空编辑器"
                >
                  <Trash2 className="size-3.5" />
                </Button>
                <Button
                  size="sm"
                  className="h-7 px-3 gap-1 text-[11px]"
                  onClick={handleExecuteSQL}
                  disabled={executingSQL || !sqlQuery.trim()}
                >
                  <Play className="size-3" />
                  {executingSQL ? "运行中..." : "运行"}
                </Button>
              </div>
            </div>
          </div>

          {/* 上半区：编辑器输入框 */}
          <div
            style={{ height: editorHeight }}
            className="w-full overflow-hidden relative min-h-[100px] bg-background"
          >
            <CodeMirror
              value={sqlQuery}
              height="100%"
              extensions={[sql()]}
              theme={cmTheme}
              onChange={(value) => setSqlQuery(value)}
              className="h-full text-xs font-mono"
              basicSetup={{
                lineNumbers: true,
                foldGutter: true,
                dropCursor: true,
                allowMultipleSelections: false,
                indentOnInput: true,
              }}
            />
          </div>

          {/* 拖动分割线 */}
          <div
            onMouseDown={handleMouseDown}
            className="h-1.5 bg-border/60 hover:bg-primary/50 cursor-row-resize transition-colors flex items-center justify-center shrink-0 select-none z-10"
            title="拖动调整大小"
          >
            <div className="w-8 h-1 rounded bg-muted-foreground/30" />
          </div>

          {/* 下半区：查看结果 (可滑动查看) */}
          <div className="flex-1 min-h-[100px] flex flex-col overflow-hidden bg-muted/10">
            {/* 结果栏工具提示 */}
            <div className="flex items-center justify-between px-4 py-1.5 border-b bg-muted/20 shrink-0 text-[11px] text-muted-foreground font-mono">
              <span className="font-semibold">执行输出</span>
              {sqlResult && (
                <span>
                  类型: {sqlResult.type.toUpperCase()} | 耗时: {sqlResult.execution_time_ms} ms
                </span>
              )}
            </div>

            {/* 滚动结果内容 */}
            <div className="flex-1 overflow-auto p-4 min-h-0">
              {executingSQL && (
                <div className="flex flex-col items-center justify-center h-full py-10 space-y-2">
                  <RefreshCw className="size-5 text-primary animate-spin" />
                  <span className="text-xs text-muted-foreground">正在在数据库执行查询，请稍候...</span>
                </div>
              )}

              {sqlError && (
                <div className="bg-destructive/10 border border-destructive/20 text-destructive font-mono text-xs p-4 rounded-lg overflow-auto h-full max-h-[300px]">
                  <p className="font-semibold mb-1">SQL 执行报错 (Error):</p>
                  <pre className="whitespace-pre-wrap">{sqlError}</pre>
                </div>
              )}

              {sqlResult && (
                <div className="h-full flex flex-col">
                  {sqlResult.type === "select" && sqlResult.columns && sqlResult.columns.length > 0 && (
                    <div className="border rounded-md overflow-auto max-w-full max-h-full bg-background relative flex-1 min-h-0">
                      <Table className="text-xs">
                        <TableHeader className="bg-muted/40 font-semibold sticky top-0 z-10">
                          <TableRow>
                            {sqlResult.columns.map((col) => (
                              <TableHead key={col} className="font-semibold text-foreground py-2">{col}</TableHead>
                            ))}
                          </TableRow>
                        </TableHeader>
                        <TableBody>
                          {sqlResult.results && sqlResult.results.length > 0 ? (
                            sqlResult.results.map((row, rIndex) => (
                              <TableRow key={rIndex} className="hover:bg-muted/10">
                                {sqlResult.columns!.map((col) => (
                                  <TableCell key={col} className="font-mono py-1.5">
                                    <span className="truncate max-w-[240px] block" title={formatCellValue(row[col])}>
                                      {formatCellValue(row[col])}
                                    </span>
                                  </TableCell>
                                ))}
                              </TableRow>
                            ))
                          ) : (
                            <TableRow>
                              <TableCell colSpan={sqlResult.columns.length} className="text-center py-10 text-muted-foreground">
                                查询结果为空
                              </TableCell>
                            </TableRow>
                          )}
                        </TableBody>
                      </Table>
                    </div>
                  )}

                  {sqlResult.type === "exec" && (
                    <div className="bg-primary/5 border border-primary/10 text-primary-foreground font-mono text-xs p-6 rounded-lg text-center my-auto">
                      <p className="text-foreground text-sm font-semibold">SQL 执行成功</p>
                      <p className="text-muted-foreground mt-2">
                        受影响行数: {sqlResult.affected_rows} 行，耗时 {sqlResult.execution_time_ms} 毫秒。
                      </p>
                    </div>
                  )}
                </div>
              )}

              {!executingSQL && !sqlResult && !sqlError && (
                <div className="flex flex-col items-center justify-center h-full py-10 text-muted-foreground opacity-60">
                  <Terminal className="size-8 mb-2" />
                  <span className="text-xs">编辑器就绪，请在上方编写 SQL 并运行查看结果</span>
                </div>
              )}
            </div>
          </div>
        </div>
      </div>
    )
  }

  // 正常数据概览及表分页浏览器渲染
  return (
    <div className="space-y-6 p-1 w-full">
      {/* 顶部控制与标题 */}
      <div className="flex flex-col sm:flex-row sm:items-center sm:justify-between border-b border-border/30 pb-4 gap-4">
        <div>
          <h2 className="text-xl font-bold tracking-tight">数据管理</h2>
          <p className="text-xs text-muted-foreground">查看物理数据库运行运维数据，进行表级数据浏览与 SQL 交互查询</p>
        </div>
        <Button
          size="sm"
          variant="secondary"
          className="h-8 gap-1.5 text-xs self-start sm:self-auto"
          onClick={() => fetchOverview()}
          disabled={loadingOverview}
        >
          <RefreshCw className={`size-3 ${loadingOverview ? "animate-spin" : ""}`} />
          刷新数据
        </Button>
      </div>

      {/* 1. 顶部概览卡片 */}
      {loadingOverview ? (
        renderOverviewSkeleton()
      ) : (
        overview && (
          <div className="grid grid-cols-2 md:grid-cols-3 lg:grid-cols-6 gap-4">
            {/* 卡片1: 数据库类型 */}
            <Card className="shadow-sm border-border/40 bg-card/50 backdrop-blur-sm hover:border-primary/20 transition-all duration-300">
              <CardHeader className="flex flex-row items-center justify-between pb-2 space-y-0">
                <CardDescription className="text-[10px] font-medium">数据库类型</CardDescription>
                <Server className="size-3.5 text-muted-foreground" />
              </CardHeader>
              <CardContent>
                <div className="text-sm font-bold uppercase">{overview.type}</div>
              </CardContent>
            </Card>

            {/* 卡片2: 数据库版本 */}
            <Card className="shadow-sm border-border/40 bg-card/50 backdrop-blur-sm hover:border-primary/20 transition-all duration-300 col-span-1 md:col-span-2 lg:col-span-1">
              <CardHeader className="flex flex-row items-center justify-between pb-2 space-y-0">
                <CardDescription className="text-[10px] font-medium">版本信息</CardDescription>
                <Cpu className="size-3.5 text-muted-foreground" />
              </CardHeader>
              <CardContent>
                <div className="text-xs font-semibold truncate" title={overview.version}>
                  {overview.version.split(" ").slice(0, 2).join(" ")}
                </div>
              </CardContent>
            </Card>

            {/* 卡片3: 数据库名称 */}
            <Card className="shadow-sm border-border/40 bg-card/50 backdrop-blur-sm hover:border-primary/20 transition-all duration-300">
              <CardHeader className="flex flex-row items-center justify-between pb-2 space-y-0">
                <CardDescription className="text-[10px] font-medium">名称/路径</CardDescription>
                <Database className="size-3.5 text-muted-foreground" />
              </CardHeader>
              <CardContent>
                <div className="text-xs font-semibold truncate" title={overview.name}>
                  {overview.name.substring(overview.name.lastIndexOf("/") + 1)}
                </div>
              </CardContent>
            </Card>

            {/* 卡片4: 数据库大小 */}
            <Card className="shadow-sm border-border/40 bg-card/50 backdrop-blur-sm hover:border-primary/20 transition-all duration-300">
              <CardHeader className="flex flex-row items-center justify-between pb-2 space-y-0">
                <CardDescription className="text-[10px] font-medium">数据库大小</CardDescription>
                <HardDrive className="size-3.5 text-muted-foreground" />
              </CardHeader>
              <CardContent>
                <div className="text-sm font-bold">{overview.size}</div>
              </CardContent>
            </Card>

            {/* 卡片5: 表数量 */}
            <Card className="shadow-sm border-border/40 bg-card/50 backdrop-blur-sm hover:border-primary/20 transition-all duration-300">
              <CardHeader className="flex flex-row items-center justify-between pb-2 space-y-0">
                <CardDescription className="text-[10px] font-medium">物理数据表</CardDescription>
                <Layers className="size-3.5 text-muted-foreground" />
              </CardHeader>
              <CardContent>
                <div className="text-sm font-bold">{formatNumber(overview.table_count)}</div>
              </CardContent>
            </Card>

            {/* 卡片6: 连接数 */}
            <Card className="shadow-sm border-border/40 bg-card/50 backdrop-blur-sm hover:border-primary/20 transition-all duration-300">
              <CardHeader className="flex flex-row items-center justify-between pb-2 space-y-0">
                <CardDescription className="text-[10px] font-medium">活跃连接数</CardDescription>
                <Activity className="size-3.5 text-muted-foreground" />
              </CardHeader>
              <CardContent>
                <div className="text-sm font-bold">{formatNumber(overview.connections)}</div>
              </CardContent>
            </Card>
          </div>
        )
      )}

      {/* 2. 中部数据表浏览器 */}
      <Card className="border-border/40 bg-card/50 backdrop-blur-sm shadow-sm">
        <CardHeader className="pb-3 border-b border-dashed flex flex-col md:flex-row md:items-center md:justify-between gap-4">
          <div className="space-y-0.5">
            <CardTitle className="text-sm font-semibold">数据表浏览器</CardTitle>
            <CardDescription className="text-[11px]">浏览数据库中的物理数据表详情及内容</CardDescription>
          </div>

          <div className="flex items-center gap-2">
            {loadingTables ? (
              <Skeleton className="h-8 w-48" />
            ) : (
              <Select value={selectedTable} onValueChange={handleTableChange}>
                <SelectTrigger className="h-8 w-[200px] text-xs bg-background border-border/40">
                  <SelectValue placeholder="选择数据表" />
                </SelectTrigger>
                <SelectContent className="max-h-[300px]">
                  {tables.map((t) => (
                    <SelectItem key={t} value={t} className="text-xs font-mono">{t}</SelectItem>
                  ))}
                </SelectContent>
              </Select>
            )}
          </div>
        </CardHeader>
        <CardContent className="pt-4">
          {loadingData && !tableData ? (
            <div className="space-y-3 py-6">
              <Skeleton className="h-6 w-full" />
              <Skeleton className="h-10 w-full" />
              <Skeleton className="h-10 w-full" />
              <Skeleton className="h-10 w-full" />
            </div>
          ) : tableData && tableData.columns.length > 0 ? (
            <div className="space-y-4">
              {/* 数据表格区域 */}
              <div className="border rounded-md overflow-x-auto max-w-full bg-background/50 relative">
                {loadingData && (
                  <div className="absolute inset-0 bg-background/40 backdrop-blur-[1px] flex items-center justify-center z-10">
                    <RefreshCw className="size-5 text-primary animate-spin" />
                  </div>
                )}
                <Table className="text-xs">
                  <TableHeader className="bg-muted/40 font-semibold sticky top-0">
                    <TableRow>
                      {tableData.columns.map((col) => (
                        <TableHead key={col} className="font-semibold text-foreground py-2.5">{col}</TableHead>
                      ))}
                    </TableRow>
                  </TableHeader>
                  <TableBody>
                    {tableData.results && tableData.results.length > 0 ? (
                      tableData.results.map((row, rIndex) => (
                        <TableRow key={rIndex} className="hover:bg-muted/10">
                          {tableData.columns.map((col) => (
                            <TableCell key={col} className="font-mono py-2">
                              <span className="truncate max-w-[200px] block" title={formatCellValue(row[col])}>
                                {formatCellValue(row[col])}
                              </span>
                            </TableCell>
                          ))}
                        </TableRow>
                      ))
                    ) : (
                      <TableRow>
                        <TableCell colSpan={tableData.columns.length} className="text-center py-10 text-muted-foreground">
                          表中无记录数据
                        </TableCell>
                      </TableRow>
                    )}
                  </TableBody>
                </Table>
              </div>

              {/* 分页控制 */}
              <div className="flex items-center justify-between text-xs text-muted-foreground flex-wrap gap-2 pt-2 border-t border-dashed">
                <div>
                  总行数: <span className="font-mono text-foreground font-semibold">{formatNumber(tableData.total)}</span> 条记录
                </div>
                <div className="flex items-center gap-2">
                  <Button
                    variant="outline"
                    size="sm"
                    className="h-7 px-2 text-[11px]"
                    onClick={() => handlePageChange(page - 1)}
                    disabled={page <= 1 || loadingData}
                  >
                    上一页
                  </Button>
                  <span className="text-xs px-2 font-mono">
                    第 {page} / {Math.max(1, Math.ceil(tableData.total / pageSize))} 页
                  </span>
                  <Button
                    variant="outline"
                    size="sm"
                    className="h-7 px-2 text-[11px]"
                    onClick={() => handlePageChange(page + 1)}
                    disabled={page >= Math.ceil(tableData.total / pageSize) || loadingData}
                  >
                    下一页
                  </Button>
                </div>
              </div>
            </div>
          ) : (
            <div className="flex flex-col items-center justify-center py-10 text-muted-foreground">
              <Layers className="size-8 opacity-45 mb-2" />
              <span className="text-xs">未选择数据表或表结构无法加载</span>
            </div>
          )}
        </CardContent>
      </Card>

      {/* 2.5 缓存管理 */}
      <Card className="border-border/40 bg-card/50 backdrop-blur-sm shadow-sm">
        <CardHeader className="pb-3 border-b border-dashed flex flex-col md:flex-row md:items-center md:justify-between gap-4">
          <div className="space-y-0.5">
            <div className="flex items-center gap-2">
              <HardDrive className="size-4 text-primary animate-pulse" />
              <CardTitle className="text-sm font-semibold">缓存管理</CardTitle>
            </div>
            <CardDescription className="text-[11px]">管理和监控系统级磁盘缓存的资源占用、生命周期及淘汰策略</CardDescription>
          </div>

          <Button
            size="sm"
            variant="secondary"
            className="h-8 gap-1.5 text-xs self-start md:self-auto"
            onClick={() => fetchCacheStatus(true)}
            disabled={loadingCache}
          >
            <RefreshCw className={`size-3 ${loadingCache ? "animate-spin" : ""}`} />
            刷新状态
          </Button>
        </CardHeader>
        <CardContent className="pt-4">
          {loadingCache && !cacheStatus ? (
            <div className="space-y-3 py-6">
              <Skeleton className="h-6 w-full" />
              <Skeleton className="h-10 w-full" />
              <Skeleton className="h-10 w-full" />
            </div>
          ) : cacheStatus ? (
            <div className="grid grid-cols-1 lg:grid-cols-5 gap-6">
              {/* 左边：状态区 (2/5 cols) */}
              <div className="lg:col-span-2 space-y-4">
                <h4 className="text-xs font-semibold text-muted-foreground uppercase tracking-wider">运行状态</h4>
                <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
                  {/* 已占空间 */}
                  <div className="p-4 rounded-xl border border-border/40 bg-background/30 backdrop-blur-xs hover:border-primary/20 transition-all duration-300">
                    <p className="text-[10px] text-muted-foreground font-medium mb-1">已用空间</p>
                    <p className="text-xl font-bold tracking-tight text-foreground">
                      {formatBytes(cacheStatus.total_size)}
                    </p>
                  </div>
                  {/* Key数量 */}
                  <div className="p-4 rounded-xl border border-border/40 bg-background/30 backdrop-blur-xs hover:border-primary/20 transition-all duration-300">
                    <p className="text-[10px] text-muted-foreground font-medium mb-1">缓存键数量</p>
                    <p className="text-xl font-bold tracking-tight text-foreground">
                      {formatNumber(cacheStatus.keys_count)} <span className="text-xs text-muted-foreground font-normal">个文件</span>
                    </p>
                  </div>
                </div>

                {/* 存储路径 */}
                <div className="p-4 rounded-xl border border-border/40 bg-background/30 backdrop-blur-xs hover:border-primary/20 transition-all duration-300">
                  <p className="text-[10px] text-muted-foreground font-medium mb-1.5">缓存基准目录</p>
                  <code className="text-xs font-mono bg-muted/60 px-2 py-1 rounded-md block truncate" title={cacheStatus.base_path}>
                    {cacheStatus.base_path}
                  </code>
                </div>
                <Button
                  variant="secondary"
                  size="sm"
                  className="h-8 text-xs font-medium w-full"
                  onClick={() => setShowClearConfirm(true)}
                >
                  立即清空缓存
                </Button>
              </div>

              {/* 右边：配置区 (3/5 cols) */}
              <div className="lg:col-span-3 border-t lg:border-t-0 lg:border-l border-border/40 pt-6 lg:pt-0 lg:pl-6 space-y-4">
                <h4 className="text-xs font-semibold text-muted-foreground uppercase tracking-wider">策略配置</h4>
                <form onSubmit={handleSaveConfig} className="space-y-4">
                  <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
                    <div className="space-y-1.5">
                      <Label htmlFor="maxSizeMB" className="text-xs font-medium">最大容量限制 (MB)</Label>
                      <Input
                        id="maxSizeMB"
                        type="number"
                        min="1"
                        value={maxSizeMB}
                        onChange={(e) => setMaxSizeMB(e.target.value)}
                        className="h-8 text-xs bg-background/50 border-border/40"
                        placeholder="例如 100"
                        required
                      />
                      <p className="text-[9px] text-muted-foreground">当总大小超出该值时，自动触发淘汰算法</p>
                    </div>

                    <div className="space-y-1.5">
                      <Label htmlFor="ttlMinutes" className="text-xs font-medium">生存时间限制 (分钟)</Label>
                      <Input
                        id="ttlMinutes"
                        type="number"
                        min="0"
                        value={ttlMinutes}
                        onChange={(e) => setTtlMinutes(e.target.value)}
                        className="h-8 text-xs bg-background/50 border-border/40"
                        placeholder="例如 60，0 表示永不过期"
                        required
                      />
                      <p className="text-[9px] text-muted-foreground">缓存项的最长存活时间，超期后将失效被清理</p>
                    </div>
                  </div>

                  <div className="flex items-start justify-between p-4 rounded-xl border border-border/40 bg-background/20">
                    <div className="space-y-1 pr-4">
                      <Label htmlFor="lruEnabled" className="text-xs font-semibold block cursor-pointer">
                        启用 LRU 淘汰机制
                      </Label>
                      <span className="text-[10px] text-muted-foreground block">
                        在到达最大容量限制时，自动移除最久未被访问的缓存项。关闭该功能仅清理过期项。
                      </span>
                    </div>
                    <Switch
                      id="lruEnabled"
                      checked={lruEnabled}
                      onCheckedChange={setLruEnabled}
                    />
                  </div>

                  <div className="flex justify-end pt-2">
                    <Button
                      type="submit"
                      disabled={savingConfig}
                      className="h-8 text-xs px-4"
                    >
                      {savingConfig ? "正在保存..." : "保存设置"}
                    </Button>
                  </div>
                </form>
              </div>
            </div>
          ) : (
            <div className="flex flex-col items-center justify-center py-10 text-muted-foreground">
              <HardDrive className="size-8 opacity-45 mb-2" />
              <span className="text-xs">未加载到缓存状态信息</span>
            </div>
          )}
        </CardContent>
      </Card>

      {/* 2.6 清除缓存确认对话框 */}
      <Dialog open={showClearConfirm} onOpenChange={setShowClearConfirm}>
        <DialogContent className="sm:max-w-md">
          <DialogHeader>
            <DialogTitle className="text-sm font-semibold flex items-center gap-2 text-destructive">
              <Trash2 className="size-4" />
              确认清空所有缓存？
            </DialogTitle>
            <DialogDescription className="text-xs text-muted-foreground pt-1">
              该操作将彻底清空磁盘目录下的全部缓存文件（包含临时解压、处理后的图片及各种块文件），重置键数量统计为 0。此操作不可撤销，且可能导致用户拉取资源时出现一过性的响应变慢。
            </DialogDescription>
          </DialogHeader>
          <DialogFooter className="flex gap-2 justify-end mt-4">
            <Button
              variant="outline"
              size="sm"
              className="h-8 text-xs"
              onClick={() => setShowClearConfirm(false)}
              disabled={clearingCache}
            >
              取消
            </Button>
            <Button
              variant="destructive"
              size="sm"
              className="h-8 text-xs"
              onClick={handleClearCache}
              disabled={clearingCache}
            >
              {clearingCache ? "清理中..." : "确认清空"}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* 3. 底部功能区 */}
      <Card className="border-border/40 bg-card/50 backdrop-blur-sm shadow-sm">
        <CardHeader className="pb-3 border-b border-dashed">
          <CardTitle className="text-sm font-semibold">功能区</CardTitle>
          <CardDescription className="text-[11px]">数据库导出及自定义高级 SQL 执行终端</CardDescription>
        </CardHeader>
        <CardContent className="pt-4">
          <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
            {/* 功能一: 数据库导出 */}
            <div className="flex items-center justify-between p-4 border border-dashed rounded-lg hover:bg-muted/20 transition-colors duration-150">
              <div className="space-y-1 pr-4">
                <p className="text-xs font-semibold flex items-center gap-1.5">
                  <Download className="size-4 text-primary" />
                  数据库备份导出
                </p>
                <p className="text-[10px] text-muted-foreground">
                  直接导出并下载物理数据库镜像文件（SQLite 导出 .db，PostgreSQL 导出为打包的 .sql 文本文件）
                </p>
              </div>
              <Button
                size="sm"
                variant="outline"
                className="h-8 text-xs gap-1"
                onClick={handleExport}
                disabled={exporting}
              >
                <RefreshCw className={`size-3 ${exporting ? "animate-spin" : ""}`} />
                {exporting ? "正在准备..." : "开始导出"}
              </Button>
            </div>

            {/* 功能二: SQL 查询终端 */}
            <div className="flex items-center justify-between p-4 border border-dashed rounded-lg hover:bg-muted/20 transition-colors duration-150">
              <div className="space-y-1 pr-4">
                <p className="text-xs font-semibold flex items-center gap-1.5">
                  <Terminal className="size-4 text-primary" />
                  SQL 查询控台
                </p>
                <p className="text-[10px] text-muted-foreground">
                  打开在线 SQL 控制终端，执行原生的 SQL 进行数据筛选、更新、调试或性能优化
                </p>
              </div>
              <Button
                size="sm"
                className="h-8 text-xs gap-1 bg-primary text-primary-foreground hover:bg-primary/95"
                onClick={() => {
                  setShowConsole(true)
                  setSqlQuery("SELECT * FROM users LIMIT 10;")
                }}
              >
                <FileText className="size-3.5" />
                进入控台
              </Button>
            </div>
          </div>
        </CardContent>
      </Card>
    </div>
  )
}
