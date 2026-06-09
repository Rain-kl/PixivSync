"use client"

import {useEffect, useState} from "react"
import {useMutation, useQueryClient, type UseQueryResult} from "@tanstack/react-query"
import {Loader2, Mail, Server} from "lucide-react"

import {Button} from "@/components/ui/button"
import {Card, CardContent, CardDescription, CardHeader, CardTitle} from "@/components/ui/card"
import {Input} from "@/components/ui/input"
import {Label} from "@/components/ui/label"
import {Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle} from "@/components/ui/dialog"
import {AdminService} from "@/lib/services"
import type {SystemConfig} from "@/lib/services/admin"
import {toast} from "sonner"

interface SystemTabProps {
  configs: Record<string, SystemConfig>
  systemConfigsQuery: UseQueryResult<SystemConfig[], Error>
}

export function SystemTab({ configs, systemConfigsQuery }: SystemTabProps) {
  const queryClient = useQueryClient()
  const [serverAddress, setServerAddress] = useState("")
  const [smtpHost, setSmtpHost] = useState("")
  const [smtpPort, setSmtpPort] = useState("")
  const [smtpUsername, setSmtpUsername] = useState("")
  const [smtpPassword, setSmtpPassword] = useState("")
  const [smtpTestOpen, setSmtpTestOpen] = useState(false)
  const [smtpTestTo, setSmtpTestTo] = useState("")
  const [smtpTestLog, setSmtpTestLog] = useState("")
  const [smtpTestSuccess, setSmtpTestSuccess] = useState<boolean | null>(null)
  const [smtpTestError, setSmtpTestError] = useState("")

  useEffect(() => {
    if (systemConfigsQuery.data) {
      setServerAddress(configs["server_address"]?.value || "")
      setSmtpHost(configs["smtp_host"]?.value || "")
      setSmtpPort(configs["smtp_port"]?.value || "587")
      setSmtpUsername(configs["smtp_username"]?.value || "")
      setSmtpPassword(configs["smtp_password"]?.value || "")
    }
  }, [systemConfigsQuery.data, configs])

  const saveSystemMutation = useMutation({
    mutationFn: async () => {
      const currentCfg = configs["server_address"]
      await AdminService.updateSystemConfig("server_address", {
        value: serverAddress,
        description: currentCfg?.description || "服务器地址",
      })
    },
    onSuccess: async () => {
      await queryClient.invalidateQueries({ queryKey: ["admin", "system-configs"] })
      toast.success("通用配置已成功保存")
    },
    onError: (error: Error) => {
      toast.error(error.message || "保存配置失败")
    },
  })

  const handleSystemSave = (e: React.FormEvent) => {
    e.preventDefault()
    saveSystemMutation.mutate()
  }

  const saveSmtpMutation = useMutation({
    mutationFn: async () => {
      const updates = [
        { key: "smtp_host", value: smtpHost },
        { key: "smtp_port", value: smtpPort },
        { key: "smtp_username", value: smtpUsername },
        { key: "smtp_password", value: smtpPassword },
      ]

      for (const update of updates) {
        const currentCfg = configs[update.key]
        if (update.key === "smtp_password" && (update.value === "" || update.value === "******")) {
          // If already configured and sent empty or mask, skip updating it (keep existing)
          if (currentCfg && currentCfg.value === "******") {
            continue
          }
        }
        await AdminService.updateSystemConfig(update.key, {
          value: update.value,
          description: currentCfg?.description || "",
        })
      }
    },
    onSuccess: async () => {
      await queryClient.invalidateQueries({ queryKey: ["admin", "system-configs"] })
      toast.success("SMTP 邮件配置已成功保存")
    },
    onError: (error: Error) => {
      toast.error(error.message || "保存配置失败")
    },
  })

  const handleSmtpSave = (e: React.FormEvent) => {
    e.preventDefault()
    saveSmtpMutation.mutate()
  }

  const testSmtpMutation = useMutation({
    mutationFn: async () => {
      setSmtpTestLog("正在发起连接测试...\n")
      setSmtpTestSuccess(null)
      setSmtpTestError("")

      const res = await AdminService.testSMTP({
        smtp_host: smtpHost,
        smtp_port: parseInt(smtpPort, 10) || 587,
        smtp_username: smtpUsername,
        smtp_password: smtpPassword,
        to: smtpTestTo,
      })
      return res
    },
    onSuccess: (data) => {
      setSmtpTestLog(data.log)
      if (data.success) {
        setSmtpTestSuccess(true)
        toast.success("测试邮件发送成功")
      } else {
        setSmtpTestSuccess(false)
        setSmtpTestError(data.error || "发送失败，请检查配置和日志。")
        toast.error("测试邮件发送失败")
      }
    },
    onError: (error: Error) => {
      setSmtpTestSuccess(false)
      setSmtpTestError(error.message || "请求发送失败")
      setSmtpTestLog((prev) => prev + `\n[请求错误] ${error.message}\n`)
      toast.error(error.message || "测试请求发送失败")
    },
  })

  const handleSmtpTestSubmit = (e: React.FormEvent) => {
    e.preventDefault()
    if (!smtpTestTo) {
      toast.error("请输入目标邮箱地址")
      return
    }
    testSmtpMutation.mutate()
  }

  return (
    <div className="space-y-6">
      {/* 通用设置 */}
      <Card className="border border-dashed shadow-sm">
        <CardHeader className="border-b border-dashed pb-4">
          <div className="flex items-center gap-2">
            <div className="p-1.5 rounded-lg bg-indigo-500/10 text-indigo-500">
              <Server className="size-4" />
            </div>
            <div>
              <CardTitle className="text-base font-semibold">通用设置</CardTitle>
              <CardDescription className="text-xs">配置系统的全局通用参数</CardDescription>
            </div>
          </div>
        </CardHeader>
        <CardContent className="pt-6">
          <form onSubmit={handleSystemSave} className="space-y-6">
            <div className="space-y-1.5">
              <Label htmlFor="server_address" className="text-xs font-semibold">服务器地址</Label>
              <Input
                id="server_address"
                type="text"
                value={serverAddress}
                onChange={(e) => setServerAddress(e.target.value)}
                placeholder="例如: https://example.com"
                className="bg-card border-dashed text-xs"
              />
              <p className="text-[10px] text-muted-foreground leading-normal">
                这里可以编辑更改服务器地址。默认不设定，允许从任意源（*）访问 API，此时存在跨域安全风险；如果手动设置服务器地址，CORS 允许源将更新为该地址，消除跨域安全隐患。
              </p>
            </div>
            <div className="flex justify-end pt-4 border-t border-dashed">
              <Button
                type="submit"
                size="sm"
                disabled={saveSystemMutation.isPending}
              >
                {saveSystemMutation.isPending ? (
                  <>
                    <Loader2 className="mr-1.5 size-3.5 animate-spin" />
                    保存中...
                  </>
                ) : (
                  "保存配置"
                )}
              </Button>
            </div>
          </form>
        </CardContent>
      </Card>

      {/* SMTP 邮件设置 */}
      <Card className="border border-dashed shadow-sm">
        <CardHeader className="border-b border-dashed pb-4">
          <div className="flex items-center gap-2">
            <div className="p-1.5 rounded-lg bg-indigo-500/10 text-indigo-500">
              <Mail className="size-4" />
            </div>
            <div>
              <CardTitle className="text-base font-semibold">SMTP 邮件设置</CardTitle>
              <CardDescription className="text-xs">配置系统的邮件发送服务 (SMTP)</CardDescription>
            </div>
          </div>
        </CardHeader>
        <CardContent className="pt-6">
          <form onSubmit={handleSmtpSave} className="space-y-6">
            <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
              <div className="space-y-1.5">
                <Label htmlFor="smtp_host" className="text-xs font-semibold">SMTP 服务器地址</Label>
                <Input
                  id="smtp_host"
                  type="text"
                  value={smtpHost}
                  onChange={(e) => setSmtpHost(e.target.value)}
                  placeholder="例如: smtp.example.com"
                  className="bg-card border-dashed text-xs"
                />
              </div>

              <div className="space-y-1.5">
                <Label htmlFor="smtp_port" className="text-xs font-semibold">SMTP 端口</Label>
                <Input
                  id="smtp_port"
                  type="number"
                  value={smtpPort}
                  onChange={(e) => setSmtpPort(e.target.value)}
                  placeholder="例如: 587 或 465"
                  className="bg-card border-dashed text-xs"
                />
              </div>

              <div className="space-y-1.5">
                <Label htmlFor="smtp_username" className="text-xs font-semibold">SMTP 账户</Label>
                <Input
                  id="smtp_username"
                  type="text"
                  value={smtpUsername}
                  onChange={(e) => setSmtpUsername(e.target.value)}
                  placeholder="例如: sender@example.com"
                  className="bg-card border-dashed text-xs"
                />
              </div>

              <div className="space-y-1.5">
                <Label htmlFor="smtp_password" className="text-xs font-semibold">SMTP 访问凭证</Label>
                <Input
                  id="smtp_password"
                  type="password"
                  value={smtpPassword}
                  onChange={(e) => setSmtpPassword(e.target.value)}
                  placeholder={configs["smtp_password"]?.value === "******" ? "•••••• (已配置，留空或输入新值)" : "输入凭证密码"}
                  className="bg-card border-dashed text-xs"
                />
              </div>
            </div>

            <div className="flex justify-end gap-2 pt-4 border-t border-dashed">
              <Button
                type="button"
                variant="outline"
                size="sm"
                onClick={() => {
                  setSmtpTestOpen(true)
                  setSmtpTestTo("")
                  setSmtpTestLog("")
                  setSmtpTestSuccess(null)
                  setSmtpTestError("")
                }}
                disabled={saveSmtpMutation.isPending}
              >
                测试发件
              </Button>
              <Button
                type="submit"
                size="sm"
                disabled={saveSmtpMutation.isPending}
              >
                {saveSmtpMutation.isPending ? (
                  <>
                    <Loader2 className="mr-1.5 size-3.5 animate-spin" />
                    保存中...
                  </>
                ) : (
                  "保存配置"
                )}
              </Button>
            </div>
          </form>
        </CardContent>
      </Card>

      <Dialog open={smtpTestOpen} onOpenChange={setSmtpTestOpen}>
        <DialogContent className="max-w-lg border border-dashed">
          <DialogHeader>
            <DialogTitle className="text-base font-semibold">SMTP 发件测试</DialogTitle>
            <DialogDescription className="text-xs">
              输入接收测试邮件的邮箱地址。系统将使用您在表单中当前填写的 SMTP 配置进行发件测试。
            </DialogDescription>
          </DialogHeader>

          <form onSubmit={handleSmtpTestSubmit} className="space-y-4">
            <div className="space-y-1.5">
              <Label htmlFor="smtp_test_to" className="text-xs font-semibold">目标邮箱地址</Label>
              <Input
                id="smtp_test_to"
                type="email"
                required
                value={smtpTestTo}
                onChange={(e) => setSmtpTestTo(e.target.value)}
                placeholder="例如: receiver@example.com"
                className="bg-card border-dashed text-xs"
                disabled={testSmtpMutation.isPending}
              />
            </div>

            {smtpTestLog && (
              <div className="space-y-1.5">
                <Label className="text-xs font-semibold">连接与传输日志</Label>
                <pre className="bg-zinc-950 text-zinc-50 font-mono p-4 rounded-lg text-[10px] h-60 overflow-y-auto whitespace-pre-wrap border border-dashed border-zinc-800 leading-relaxed">
                  {smtpTestLog}
                </pre>
              </div>
            )}

            {smtpTestSuccess === true && (
              <div className="p-3 rounded-lg border border-dashed border-emerald-500/30 bg-emerald-500/5 text-emerald-500 text-xs">
                测试成功！邮件已顺利发出。
              </div>
            )}

            {smtpTestSuccess === false && (
              <div className="p-3 rounded-lg border border-dashed border-rose-500/30 bg-rose-500/5 text-rose-500 text-xs break-all">
                测试失败：{smtpTestError}
              </div>
            )}

            <DialogFooter className="gap-2 sm:gap-0 border-t border-dashed pt-4">
              <Button
                type="button"
                variant="ghost"
                size="sm"
                onClick={() => setSmtpTestOpen(false)}
                disabled={testSmtpMutation.isPending}
              >
                关闭
              </Button>
              <Button
                type="submit"
                size="sm"
                disabled={testSmtpMutation.isPending}
              >
                {testSmtpMutation.isPending ? (
                  <>
                    <Loader2 className="mr-1.5 size-3.5 animate-spin" />
                    测试中...
                  </>
                ) : (
                  "开始测试"
                )}
              </Button>
            </DialogFooter>
          </form>
        </DialogContent>
      </Dialog>
    </div>
  )
}
