"use client"

import * as React from "react"
import {motion} from "motion/react"

import {UserFileManager} from "./file-manager"

export function UserFilesMain() {
  return (
    <motion.div
      initial={{ opacity: 0, y: 15 }}
      animate={{ opacity: 1, y: 0 }}
      transition={{ duration: 0.35, ease: "easeOut" }}
      className="flex w-full flex-col gap-6 py-6"
    >
      {/* 顶部标题区 */}
      <div className="flex flex-col md:flex-row md:items-center justify-between gap-4 border-b pb-5">
        <div>
          <h1 className="text-xl font-bold tracking-tight bg-gradient-to-r from-foreground via-foreground/90 to-muted-foreground bg-clip-text text-transparent">
            我的文件
          </h1>
          <p className="text-sm text-muted-foreground">
            管理您上传的个人文件，支持上传、下载、搜索与删除操作
          </p>
        </div>
      </div>

      <UserFileManager />
    </motion.div>
  )
}
