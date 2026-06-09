"use client"

import {SystemConfigs} from "@/components/common/admin/system"
import {AdminProvider} from "@/contexts/admin-context"

/* 系统配置页面 */
export default function SystemConfigPage() {
  return (
    <AdminProvider>
      <SystemConfigs />
    </AdminProvider>
  )
}

