"use client"

import {Info, Server} from "lucide-react"
import packageJson from "../../../package.json"
import {APP_BUILD_DATE, APP_VERSION} from "@/lib/app-info"
import {apiConfig} from "@/lib/services"
import {Card, CardContent, CardDescription, CardHeader, CardTitle} from "@/components/ui/card"

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
          <InfoRow label="版本号" value={APP_VERSION} />
          {APP_BUILD_DATE && <InfoRow label="构建时间" value={APP_BUILD_DATE} />}
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
    </div>
  )
}
