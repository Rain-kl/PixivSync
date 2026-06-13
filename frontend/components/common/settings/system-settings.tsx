"use client"

import {useEffect, useMemo} from "react"
import {useQuery} from "@tanstack/react-query"
import {Loader2, Settings} from "lucide-react"
import {useRouter} from "next/navigation"
import {motion} from "motion/react"

import {Tabs, TabsContent, TabsList, TabsTrigger} from "@/components/ui/tabs"
import {useAuth} from "@/components/providers/auth-provider"
import {OperationTab} from "./operation-tab"
import {AdminService} from "@/lib/services"
import type {SystemConfig} from "@/lib/services/admin"
import {SystemStatusManager} from "@/components/common/admin/status"

import {SecurityTab} from "./security-tab"
import {SystemTab} from "./system-tab"
import {OtherTab} from "./other-tab"
import {InfoTab} from "./info-tab"

function systemConfigMap(configs: SystemConfig[]) {
  return configs.reduce<Record<string, SystemConfig>>((accumulator, config) => {
    accumulator[config.key] = config
    return accumulator
  }, {})
}

export function SystemSettingsMain() {
  const { user, loading } = useAuth()
  const router = useRouter()

  const systemConfigsQuery = useQuery({
    queryKey: ["admin", "system-configs"],
    queryFn: () => AdminService.listSystemConfigs("system"),
    enabled: !!user?.is_admin,
  })

  const configs = useMemo(
    () => systemConfigMap(systemConfigsQuery.data ?? []),
    [systemConfigsQuery.data],
  )

  useEffect(() => {
    if (!loading && (!user || !user.is_admin)) {
      router.replace("/settings/profile")
    }
  }, [user, loading, router])

  if (loading || !user || !user.is_admin) {
    return (
      <div className="flex items-center justify-center min-h-[400px]">
        <Loader2 className="size-6 animate-spin text-indigo-500" />
      </div>
    )
  }

  return (

    <motion.div
      initial={{ opacity: 0, y: 15 }}
      animate={{ opacity: 1, y: 0 }}
      transition={{ duration: 0.35, ease: "easeOut" }}
      className="py-6 space-y-6 w-full"
    >
      <div className="flex items-center gap-2">
        <Settings className="size-5 text-muted-foreground" />
        <div>
          <h1 className="text-2xl font-semibold tracking-tight">系统设置</h1>
        </div>
      </div>
      <Tabs defaultValue="security" className="w-full">
        <TabsList variant="line" className="w-fit inline-flex gap-8">
          <TabsTrigger value="security" className="px-0 pb-2 text-xs font-semibold">
            安全设置
          </TabsTrigger>
          <TabsTrigger value="operation" className="px-0 pb-2 text-xs font-semibold">
            业务设置
          </TabsTrigger>
          <TabsTrigger value="system" className="px-0 pb-2 text-xs font-semibold">
            系统设置
          </TabsTrigger>
          <TabsTrigger value="other" className="px-0 pb-2 text-xs font-semibold">
            其他设置
          </TabsTrigger>
          <TabsTrigger value="status" className="px-0 pb-2 text-xs font-semibold">
            系统状态
          </TabsTrigger>
          <TabsTrigger value="info" className="px-0 pb-2 text-xs font-semibold">
            系统信息
          </TabsTrigger>
        </TabsList>

        <TabsContent value="security" className="pt-4">
          <SecurityTab configs={configs} systemConfigsQuery={systemConfigsQuery} />
        </TabsContent>
        <TabsContent value="operation" className="pt-4">
          <OperationTab configs={configs} systemConfigsQuery={systemConfigsQuery} />
        </TabsContent>
        <TabsContent value="system" className="pt-4">
          <SystemTab configs={configs} systemConfigsQuery={systemConfigsQuery} />
        </TabsContent>
        <TabsContent value="status" className="pt-4">
          <SystemStatusManager />
        </TabsContent>
        <TabsContent value="other" className="pt-4">
          <OtherTab configs={configs} />
        </TabsContent>
        <TabsContent value="info" className="pt-4">
          <InfoTab />
        </TabsContent>
      </Tabs>
    </motion.div>
  )
}
