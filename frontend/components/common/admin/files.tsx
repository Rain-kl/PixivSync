"use client"

import * as React from "react"
import {motion} from "motion/react"

import {Tabs, TabsContent, TabsList, TabsTrigger} from "@/components/ui/tabs"
import {FileStats} from "./file-stats"
import {FileList} from "./file-list"

export function FilesMain() {
  const [activeTab, setActiveTab] = React.useState("stats")

  return (
    <motion.div
      initial={{ opacity: 0, y: 15 }}
      animate={{ opacity: 1, y: 0 }}
      transition={{ duration: 0.35, ease: "easeOut" }}
      className="py-6 space-y-6 w-full"
    >
      {/* 顶部标题区 */}
      <div className="flex flex-col md:flex-row md:items-center justify-between gap-4 border-b pb-5">
        <div>
          <h1 className="text-xl font-bold tracking-tight bg-gradient-to-r from-foreground via-foreground/90 to-muted-foreground bg-clip-text text-transparent">
            文件管理
          </h1>
          <p className="text-sm text-muted-foreground">
            管理您上传的所有文件，支持下载、数据统计与批量操作
          </p>
        </div>
      </div>

      <Tabs value={activeTab} onValueChange={setActiveTab} className="w-full space-y-6">
        <TabsList className="grid w-fit grid-cols-2">
          <TabsTrigger value="stats">文件存储信息</TabsTrigger>
          <TabsTrigger value="list">文件列表</TabsTrigger>
        </TabsList>

        {/* ──────── TAB 1: 统计看板 ──────── */}
        <TabsContent value="stats" className="outline-hidden">
          <FileStats />
        </TabsContent>

        {/* ──────── TAB 2: 文件列表 ──────── */}
        <TabsContent value="list" className="outline-hidden">
          <FileList />
        </TabsContent>
      </Tabs>
    </motion.div>
  )
}
