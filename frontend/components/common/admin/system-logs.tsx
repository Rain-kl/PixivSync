// Copyright 2026 Arctel.net
// SPDX-License-Identifier: AGPL-3.0-only

"use client"

import {Activity, BarChart3, Terminal} from "lucide-react"

import {Tabs, TabsContent, TabsList, TabsTrigger} from "@/components/ui/tabs"
import {AppLogs} from "./app-logs"
import {AccessLogs} from "./access-logs"
import {AccessAnalytics} from "./access-analytics"

export function SystemLogs() {
  return (
    <div className="flex flex-col h-full space-y-4 py-6">
      {/* Header */}
      <div className="flex items-center gap-2">
        <Terminal className="size-5 text-muted-foreground" />
        <div>
          <h1 className="text-2xl font-semibold tracking-tight">系统日志</h1>
        </div>
      </div>
      {/* Tabs Layout */}
      <Tabs defaultValue="analytics" className="w-full flex-grow flex flex-col gap-2">
        <TabsList variant="line">
          <TabsTrigger value="analytics" className="flex items-center gap-1.5 px-4">
            <BarChart3 className="size-3.5" />
            访问分析
          </TabsTrigger>
          <TabsTrigger value="access" className="flex items-center gap-1.5 px-4">
            <Activity className="size-3.5" />
            用户访问日志
          </TabsTrigger>
          <TabsTrigger value="app" className="flex items-center gap-1.5 px-4">
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

