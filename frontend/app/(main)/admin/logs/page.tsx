/*
Copyright 2026 Arctel.net

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

"use client"

import {Activity, BarChart3, Terminal} from "lucide-react"

import {Tabs, TabsContent, TabsList, TabsTrigger} from "@/components/ui/tabs"
import {AppLogs} from "@/components/common/admin/app-logs"
import {AccessLogs} from "@/components/common/admin/access-logs"
import {AccessAnalytics} from "@/components/common/admin/access-analytics"

export default function LogsPage() {
  return (
    <div className="flex flex-col h-full space-y-6 py-6">
      {/* Header */}
      <div className="flex items-center gap-2">
        <Terminal className="size-5 text-primary" />
        <div>
          <h1 className="text-2xl font-semibold tracking-tight">系统日志</h1>
        </div>
      </div>
      {/* Tabs Layout */}
      <Tabs defaultValue="analytics" className="w-full">
        <TabsList variant="line" className="w-fit inline-flex gap-8 mb-6">
          <TabsTrigger value="analytics" className="px-0 pb-2 text-xs font-semibold flex items-center gap-1.5">
            <BarChart3 className="size-3.5" />
            访问分析
          </TabsTrigger>
          <TabsTrigger value="access" className="px-0 pb-2 text-xs font-semibold flex items-center gap-1.5">
            <Activity className="size-3.5" />
            用户访问日志
          </TabsTrigger>
          <TabsTrigger value="app" className="px-0 pb-2 text-xs font-semibold flex items-center gap-1.5">
            <Terminal className="size-3.5" />
            应用运行日志
          </TabsTrigger>
        </TabsList>

        <TabsContent value="analytics" className="mt-0 outline-none flex-1">
          <AccessAnalytics />
        </TabsContent>
        <TabsContent value="access" className="mt-0 outline-none flex-1">
          <AccessLogs />
        </TabsContent>
        <TabsContent value="app" className="mt-0 outline-none flex-1">
          <AppLogs />
        </TabsContent>
      </Tabs>
    </div>
  )
}

