"use client"

import {Suspense} from "react"
import {useRouter, useSearchParams} from "next/navigation"
import {Tabs, TabsContent, TabsList, TabsTrigger} from "@/components/ui/tabs"
import {TaskManager} from "@/components/common/admin/task-manager"
import {TaskSchedulesManager} from "@/components/common/admin/task-schedules"
import {TaskExecutionsManager} from "@/components/common/admin/task-executions"
import {Spinner} from "@/components/ui/spinner"
import {Activity} from "lucide-react"

function TasksPageContent() {
  const searchParams = useSearchParams()
  const router = useRouter()
  const activeTab = searchParams.get("tab") || "tasks"

  const handleTabChange = (value: string) => {
    router.push(`/admin/tasks?tab=${value}`)
  }
  return (
    <div className="py-6 space-y-6">
      <div className="flex items-center gap-3 pb-2">
        <div className="p-2 bg-primary/10 rounded-lg text-primary">
          <Activity className="size-5" />
        </div>
        <div>
          <h1 className="text-2xl font-bold tracking-tight">任务管理</h1>
          <p className="text-xs text-muted-foreground mt-0.5">
            下发异步镜像与同步导出任务，管理后台执行计划及调度日志。
          </p>
        </div>
      </div>

      <Tabs value={activeTab} onValueChange={handleTabChange} className="w-full">
        <TabsList variant="line">
          <TabsTrigger value="tasks">任务管理</TabsTrigger>
          <TabsTrigger value="schedules">定时任务</TabsTrigger>
          <TabsTrigger value="executions">任务日志</TabsTrigger>
        </TabsList>
        <TabsContent value="tasks" className="space-y-4 outline-none">
          <TaskManager />
        </TabsContent>
        <TabsContent value="schedules" className="space-y-4 outline-none">
          <TaskSchedulesManager />
        </TabsContent>
        <TabsContent value="executions" className="space-y-4 outline-none">
          <TaskExecutionsManager />
        </TabsContent>
      </Tabs>
    </div>
  )
}

export default function TasksPage() {
  return (
    <Suspense fallback={
      <div className="flex items-center justify-center min-h-[400px]">
        <Spinner className="h-8 w-8" />
      </div>
    }>
      <TasksPageContent />
    </Suspense>
  )
}
