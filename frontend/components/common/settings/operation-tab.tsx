"use client"

import {useMutation, useQueryClient, type UseQueryResult} from "@tanstack/react-query"
import {Globe, Search} from "lucide-react"
import {Card, CardContent, CardDescription, CardHeader, CardTitle} from "@/components/ui/card"
import {Switch} from "@/components/ui/switch"
import {AdminService} from "@/lib/services"
import type {SystemConfig} from "@/lib/services/admin"
import {TemplatesManager} from "./templates"
import {toast} from "sonner"

interface OperationTabProps {
  configs: Record<string, SystemConfig>
  systemConfigsQuery: UseQueryResult<SystemConfig[], Error>
}

export function OperationTab({ configs, systemConfigsQuery }: OperationTabProps) {
  const queryClient = useQueryClient()

  const updateConfigMutation = useMutation({
    mutationFn: async ({ key, value }: { key: string; value: boolean }) => {
      const config = configs[key]
      if (!config) {
        throw new Error(`缺少配置项: ${key}`)
      }
      await AdminService.updateSystemConfig(key, {
        value: value ? "true" : "false",
        description: config.description,
      })
    },
    onSuccess: async () => {
      await queryClient.invalidateQueries({ queryKey: ["admin", "system-configs"] })
      await queryClient.invalidateQueries({ queryKey: ["public-config"] })
      toast.success("运营配置已更新")
    },
    onError: (error: Error) => {
      toast.error(error.message || "更新配置失败")
    },
  })

  const indexingEnabled = configs["search_engine_indexing_enabled"]?.value === "true"

  return (
    <div className="space-y-6">
      {/* 搜索引擎检索设置 */}
      <Card className="border border-dashed shadow-sm">
        <CardHeader className="border-b border-dashed pb-4">
          <div className="flex items-center gap-2">
            <div className="p-1.5 rounded-lg bg-indigo-500/10 text-indigo-500">
              <Globe className="size-4" />
            </div>
            <div>
              <CardTitle className="text-base font-semibold">SEO 与搜索引擎检索</CardTitle>
              <CardDescription className="text-xs">配置站点是否允许被搜索引擎抓取和检索</CardDescription>
            </div>
          </div>
        </CardHeader>
        <CardContent className="pt-6">
          <div className="flex items-center justify-between gap-4 rounded-xl border border-dashed p-4 bg-card hover:bg-muted/10 hover:border-indigo-500/30 transition-all duration-300 shadow-sm">
            <div className="space-y-1">
              <div className="flex items-center gap-2">
                <Search className="size-4 text-indigo-500" />
                <span className="font-medium text-sm text-foreground">允许被搜索引擎检索</span>
              </div>
              <p className="text-xs text-muted-foreground leading-relaxed pr-2">
                默认关闭。关闭后系统将自动下发 noindex 指令并限制 robots.txt，禁止搜索引擎爬虫抓取本站页面。
              </p>
            </div>
            <Switch
              checked={indexingEnabled}
              disabled={updateConfigMutation.isPending || systemConfigsQuery.isPending}
              onCheckedChange={(checked) =>
                updateConfigMutation.mutate({ key: "search_engine_indexing_enabled", value: checked })
              }
            />
          </div>
        </CardContent>
      </Card>

      {/* 通知模板管理 */}
      <TemplatesManager />
    </div>
  )
}
