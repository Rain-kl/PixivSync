"use client"

import {useState} from "react"
import {Database, Download, Info, Loader2, Server} from "lucide-react"
import packageJson from "../../../package.json"
import {AdminService, apiConfig} from "@/lib/services"
import {Card, CardContent, CardDescription, CardHeader, CardTitle} from "@/components/ui/card"
import {Button} from "@/components/ui/button"
import {Badge} from "@/components/ui/badge"
import {useQuery} from "@tanstack/react-query"
import {toast} from "sonner"

function InfoRow({ label, value }: { label: string; value: React.ReactNode }) {
  return (
    <div className="flex items-center justify-between gap-4 border-b border-dashed py-2 last:border-b-0">
      <span className="text-xs text-muted-foreground">{label}</span>
      <span className="text-right text-xs font-medium text-foreground break-all">{value || "-"}</span>
    </div>
  )
}

interface InfoTabProps {
  systemConfigsLength: number
  authSourcesLength: number
}

export function InfoTab({ systemConfigsLength, authSourcesLength }: InfoTabProps) {
  const [exporting, setExporting] = useState(false)

  const dbInfoQuery = useQuery({
    queryKey: ["admin", "db-info"],
    queryFn: () => AdminService.getDatabaseInfo(),
  })

  const dbInfo = dbInfoQuery.data

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
      toast.success("数据库导出成功", { description: `文件已下载：${filename}` })
    } catch (err) {
      toast.error("数据库导出失败", {
        description: err instanceof Error ? err.message : "请求时发生未知错误",
      })
    } finally {
      setExporting(false)
    }
  }

  return (
    <div className="grid grid-cols-1 gap-4 lg:grid-cols-2">
      <Card className="border border-dashed shadow-sm">
        <CardHeader className="border-b border-dashed pb-4">
          <div className="flex items-center gap-2">
            <div className="p-1.5 rounded-lg bg-muted text-muted-foreground">
              <Info className="size-4" />
            </div>
            <div>
              <CardTitle className="text-base font-semibold">应用信息</CardTitle>
              <CardDescription className="text-xs">当前前端应用的版本与构建信息</CardDescription>
            </div>
          </div>
        </CardHeader>
        <CardContent className="pt-4">
          <InfoRow label="应用名称" value={packageJson.name} />
          <InfoRow label="版本号" value={packageJson.version} />
          <InfoRow label="构建时间" value={packageJson.buildDate} />
          <InfoRow label="Next.js" value={(packageJson as { dependencies?: Record<string, string> }).dependencies?.next} />
          <InfoRow label="React" value={(packageJson as { dependencies?: Record<string, string> }).dependencies?.react} />
        </CardContent>
      </Card>

      <Card className="border border-dashed shadow-sm">
        <CardHeader className="border-b border-dashed pb-4">
          <div className="flex items-center gap-2">
            <div className="p-1.5 rounded-lg bg-muted text-muted-foreground">
              <Server className="size-4" />
            </div>
            <div>
              <CardTitle className="text-base font-semibold">服务连接</CardTitle>
              <CardDescription className="text-xs">前端 API 客户端的基础连接参数</CardDescription>
            </div>
          </div>
        </CardHeader>
        <CardContent className="pt-4">
          <InfoRow label="API Base URL" value={apiConfig.baseURL || "同源"} />
          <InfoRow label="请求超时" value={`${apiConfig.timeout}ms`} />
          <InfoRow label="携带凭证" value={apiConfig.withCredentials ? "是" : "否"} />
          <InfoRow label="系统配置项" value={`${systemConfigsLength} 项`} />
          <InfoRow label="认证源数量" value={`${authSourcesLength} 个`} />
        </CardContent>
      </Card>

      {/* 数据库信息卡片 */}
      <Card className="border border-dashed shadow-sm lg:col-span-2">
        <CardHeader className="border-b border-dashed pb-4">
          <div className="flex items-center justify-between">
            <div className="flex items-center gap-2">
              <div className="p-1.5 rounded-lg bg-muted text-muted-foreground">
                <Database className="size-4" />
              </div>
              <div>
                <CardTitle className="text-base font-semibold">数据库信息</CardTitle>
                <CardDescription className="text-xs">当前使用的数据库类型及连接信息</CardDescription>
              </div>
            </div>
            <Button
              id="db-export-btn"
              size="sm"
              variant="outline"
              className="h-8 text-xs gap-1.5"
              onClick={handleExport}
              disabled={exporting || dbInfoQuery.isLoading}
            >
              {exporting ? (
                <Loader2 className="size-3 animate-spin" />
              ) : (
                <Download className="size-3" />
              )}
              {exporting ? "导出中..." : "数据导出"}
            </Button>
          </div>
        </CardHeader>
        <CardContent className="pt-4">
          {dbInfoQuery.isLoading ? (
            <div className="flex items-center gap-2 py-2 text-xs text-muted-foreground">
              <Loader2 className="size-3 animate-spin" />
              加载数据库信息中...
            </div>
          ) : dbInfoQuery.isError ? (
            <p className="text-xs text-destructive py-2">获取数据库信息失败</p>
          ) : (
            <>
              <div className="flex items-center justify-between gap-4 border-b border-dashed py-2">
                <span className="text-xs text-muted-foreground">数据库类型</span>
                <Badge
                  variant="secondary"
                  className={
                    dbInfo?.type === "postgres"
                      ? "bg-blue-500/10 text-blue-600 dark:bg-blue-500/20 dark:text-blue-400 border-none text-[11px]"
                      : "bg-amber-500/10 text-amber-600 dark:bg-amber-500/20 dark:text-amber-400 border-none text-[11px]"
                  }
                >
                  {dbInfo?.type === "postgres" ? "PostgreSQL" : "SQLite"}
                </Badge>
              </div>
              <InfoRow
                label={dbInfo?.type === "postgres" ? "数据库名称" : "数据库文件"}
                value={dbInfo?.name}
              />
              <InfoRow label="版本信息" value={dbInfo?.version} />
            </>
          )}
        </CardContent>
      </Card>
    </div>
  )
}
