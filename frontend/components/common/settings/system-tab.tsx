"use client"

import {useEffect, useState} from "react"
import {useMutation, useQueryClient, type UseQueryResult} from "@tanstack/react-query"
import {Globe, Info, Loader2, Mail, Search, Server, Sparkles} from "lucide-react"

import {Button} from "@/components/ui/button"
import {Card, CardContent, CardDescription, CardHeader, CardTitle} from "@/components/ui/card"
import {Input} from "@/components/ui/input"
import {Label} from "@/components/ui/label"
import {Switch} from "@/components/ui/switch"
import {Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle} from "@/components/ui/dialog"
import {Badge} from "@/components/ui/badge"
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

  const handleDetectAddress = () => {
    if (typeof window !== "undefined") {
      setServerAddress(window.location.origin)
      toast.success("已自动获取当前域名并填充")
    }
  }

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
      toast.success("配置已更新")
    },
    onError: (error: Error) => {
      toast.error(error.message || "更新配置失败")
    },
  })

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

  const indexingEnabled = configs["search_engine_indexing_enabled"]?.value === "true"

  return (
    <div className="space-y-8">
      {/* 通用设置 */}
      <Card className="border border-zinc-200 dark:border-zinc-800 shadow-md bg-gradient-to-b from-card to-zinc-50/30 dark:to-zinc-950/20 overflow-hidden">
        <CardHeader className="border-b border-zinc-100 dark:border-zinc-800 pb-5 bg-zinc-50/50 dark:bg-zinc-900/30">
          <div className="flex items-center gap-3">
            <div className="p-2.5 rounded-xl bg-indigo-500/10 text-indigo-500 ring-4 ring-indigo-500/5">
              <Server className="size-5" />
            </div>
            <div>
              <CardTitle className="text-lg font-bold tracking-tight">通用设置</CardTitle>
              <CardDescription className="text-xs text-muted-foreground">配置系统的全局网络通信访问限制与搜索引擎的公开检索收录参数</CardDescription>
            </div>
          </div>
        </CardHeader>
        <CardContent className="p-6">
          <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">

            {/* 跨域源与服务器地址 */}
            <form onSubmit={handleSystemSave} className="group relative flex flex-col justify-between space-y-5 rounded-2xl border border-zinc-200/80 dark:border-zinc-800/80 p-6 bg-card hover:shadow-lg hover:border-indigo-500/30 dark:hover:border-indigo-500/30 transition-all duration-300">
              <div className="space-y-4">
                <div className="flex items-center justify-between">
                  <div className="flex items-center gap-2.5 text-indigo-500">
                    <Globe className="size-5" />
                    <span className="font-semibold text-sm text-foreground">访问域名与跨域来源限制</span>
                  </div>
                  <Button
                    type="button"
                    variant="outline"
                    size="sm"
                    onClick={handleDetectAddress}
                    className="h-7 px-2.5 text-[10px] gap-1 font-medium hover:bg-indigo-500/10 hover:text-indigo-500 hover:border-indigo-500/30 transition-all duration-200 shadow-xs"
                  >
                    <Sparkles className="size-3 text-indigo-500" />
                    使用当前域名
                  </Button>
                </div>
                <div className="space-y-2">
                  <div className="flex items-center justify-between">
                    <Label htmlFor="server_address" className="text-xs font-semibold text-muted-foreground">服务器访问地址 (Server Address)</Label>
                  </div>
                  <div className="relative">
                    <Input
                      id="server_address"
                      type="text"
                      value={serverAddress}
                      onChange={(e) => setServerAddress(e.target.value)}
                      placeholder="例如: https://example.com"
                      className="bg-zinc-50/50 dark:bg-zinc-900/50 border-zinc-200 dark:border-zinc-800 text-xs focus-visible:ring-1 focus-visible:ring-indigo-500 transition-all duration-200 h-9.5"
                    />
                  </div>
                  <div className="flex items-start gap-1.5 mt-2 bg-zinc-50 dark:bg-zinc-900/30 p-2.5 rounded-lg border border-zinc-100 dark:border-zinc-800/50">
                    <Info className="size-3.5 text-muted-foreground shrink-0 mt-0.5" />
                    <p className="text-[10px] text-muted-foreground leading-relaxed">
                      配置 API 的对外服务访问域名。留空则允许任意源访问（CORS 将开放 `*`，有安全风险）。若配置具体域名，将激活同源及 CORS 白名单验证保护。
                    </p>
                  </div>
                </div>
              </div>
              <div className="flex justify-end pt-4 border-t border-zinc-100 dark:border-zinc-800/50">
                <Button
                  type="submit"
                  size="sm"
                  className="h-8.5 px-4 text-xs font-medium bg-indigo-600 hover:bg-indigo-500 text-white shadow-md hover:shadow-indigo-500/10 active:scale-[0.98] transition-all"
                  disabled={saveSystemMutation.isPending}
                >
                  {saveSystemMutation.isPending ? (
                    <>
                      <Loader2 className="mr-1.5 size-3.5 animate-spin" />
                      保存中...
                    </>
                  ) : (
                    "保存访问配置"
                  )}
                </Button>
              </div>
            </form>

            {/* SEO 搜索引擎抓取 */}
            <div className={`group relative flex flex-col justify-between space-y-5 rounded-2xl border p-6 bg-card hover:shadow-lg transition-all duration-500 ${
              indexingEnabled
                ? "border-emerald-500/20 dark:border-emerald-500/30 hover:border-emerald-500/40 bg-gradient-to-b from-card to-emerald-500/[0.02] dark:to-emerald-500/[0.04]"
                : "border-zinc-200 dark:border-zinc-800 hover:border-zinc-300 dark:hover:border-zinc-700 bg-gradient-to-b from-card to-zinc-500/[0.01]"
            }`}>
              <div className="space-y-4">
                <div className="flex items-center justify-between">
                  <div className={`flex items-center gap-2.5 transition-colors duration-300 ${indexingEnabled ? "text-emerald-500" : "text-zinc-500"}`}>
                    <Search className="size-5" />
                    <span className="font-semibold text-sm text-foreground">搜索引擎抓取检索 (SEO)</span>
                  </div>
                  <Badge
                    variant={indexingEnabled ? "default" : "secondary"}
                    className={`text-[10px] font-semibold tracking-wide px-2.5 py-0.5 shadow-xs transition-all duration-300 ${
                      indexingEnabled
                        ? "bg-emerald-500/15 text-emerald-600 hover:bg-emerald-500/20 dark:bg-emerald-500/20 dark:text-emerald-400 border border-emerald-500/20"
                        : "bg-zinc-100 text-zinc-500 dark:bg-zinc-800 dark:text-zinc-400 border border-zinc-200 dark:border-zinc-700"
                    }`}
                  >
                    <span className={`inline-block size-1.5 rounded-full mr-1.5 shrink-0 ${indexingEnabled ? "bg-emerald-500 animate-pulse" : "bg-zinc-400"}`} />
                    {indexingEnabled ? "已启用索引" : "已屏蔽检索"}
                  </Badge>
                </div>
                <div className="space-y-2">
                  <span className="text-xs font-semibold text-muted-foreground block">站点检索可见性开关</span>
                  <div className="flex items-start gap-1.5 bg-zinc-50 dark:bg-zinc-900/30 p-2.5 rounded-lg border border-zinc-100 dark:border-zinc-800/50">
                    <Info className="size-3.5 text-muted-foreground shrink-0 mt-0.5" />
                    <p className="text-[10px] text-muted-foreground leading-relaxed">
                      控制本系统是否向主流搜索引擎（如 Google、Baidu、Bing）开放公开索引。关闭时，系统响应的 HTML 头部会自动注入 meta robots 标签，防止爬虫收录。
                    </p>
                  </div>
                </div>
              </div>

              <div className="flex items-center justify-between pt-4 border-t border-zinc-100 dark:border-zinc-800/50 h-12">
                <div className="flex flex-col">
                  <span className="text-xs text-foreground font-semibold">
                    {indexingEnabled ? "允许爬虫访问与收录" : "全面屏蔽外部搜索"}
                  </span>
                  <span className="text-[10px] text-muted-foreground">
                    {indexingEnabled ? "爬虫可自由搜集并抓取网站内容" : "已拦截所有网络爬虫的收录请求"}
                  </span>
                </div>
                <Switch
                  checked={indexingEnabled}
                  disabled={updateConfigMutation.isPending || systemConfigsQuery.isPending}
                  onCheckedChange={(checked) =>
                    updateConfigMutation.mutate({ key: "search_engine_indexing_enabled", value: checked })
                  }
                  className="data-[state=checked]:bg-emerald-500 dark:data-[state=checked]:bg-emerald-600 focus-visible:ring-emerald-500"
                />
              </div>
            </div>

          </div>
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

