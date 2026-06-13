"use client"

import {useState} from "react"
import {useQueryClient} from "@tanstack/react-query"
import {Chrome, Plus, RefreshCw, UsersRound, Zap} from "lucide-react"
import {toast} from "sonner"

import {Button} from "@/components/ui/button"
import {Spinner} from "@/components/ui/spinner"
import {Input} from "@/components/ui/input"
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog"
import {Tabs, TabsContent, TabsList, TabsTrigger} from "@/components/ui/tabs"
import {EmptyStateWithBorder} from "@/components/layout/empty"
import {ErrorInline} from "@/components/layout/error"
import {LoadingStateWithBorder} from "@/components/layout/loading"
import type {PixezAccount} from "@/lib/services"
import {AdminService, PixezService} from "@/lib/services"

import {AccountCard} from "./AccountCard"
import {usePixEzAccounts} from "./api/usePixEzAccounts"


export function PixEzAccounts() {
  const queryClient = useQueryClient()
  const accountsQuery = usePixEzAccounts()
  const [syncingID, setSyncingID] = useState<string | null>(null)
  const [deletingID, setDeletingID] = useState<string | null>(null)
  const [isAddOpen, setIsAddOpen] = useState(false)
  const [refreshToken, setRefreshToken] = useState("")
  const [isAdding, setIsAdding] = useState(false)

  const [codeVerifier, setCodeVerifier] = useState("")
  const [callbackUrl, setCallbackUrl] = useState("")
  const [isFetchingUrl, setIsFetchingUrl] = useState(false)

  const accounts = accountsQuery.data ?? []

  const handleOpenLogin = async () => {
    setIsFetchingUrl(true)
    try {
      const res = await PixezService.getLoginURL()
      setCodeVerifier(res.code_verifier)
      window.open(res.login_url, "_blank", "noopener,noreferrer")
      toast.success("已打开 Pixiv 登录页面")
    } catch (error) {
      toast.error("获取登录链接失败", {
        description: error instanceof Error ? error.message : "未知错误",
      })
    } finally {
      setIsFetchingUrl(false)
    }
  }

  const handleCallbackSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    if (!codeVerifier) {
      toast.error("请先点击按钮登录 Pixiv 获取 code_verifier")
      return
    }
    if (!callbackUrl.trim()) {
      toast.error("请输入回调链接或 Code")
      return
    }
    setIsAdding(true)
    try {
      await PixezService.loginCallback(callbackUrl.trim(), codeVerifier)
      toast.success("账号登录成功")
      setIsAddOpen(false)
      setCallbackUrl("")
      setCodeVerifier("")
      await queryClient.invalidateQueries({queryKey: ["pixez", "accounts"]})
    } catch (error) {
      toast.error("登录失败", {
        description: error instanceof Error ? error.message : "未知错误",
      })
    } finally {
      setIsAdding(false)
    }
  }

  const handleAddAccount = async (e: React.FormEvent) => {
    e.preventDefault()
    if (!refreshToken.trim()) {
      toast.error("请输入刷新令牌")
      return
    }
    setIsAdding(true)
    try {
      await PixezService.addAccount(refreshToken.trim())
      toast.success("账号添加成功")
      setIsAddOpen(false)
      setRefreshToken("")
      await queryClient.invalidateQueries({queryKey: ["pixez", "accounts"]})
    } catch (error) {
      toast.error("添加账号失败", {
        description: error instanceof Error ? error.message : "未知错误",
      })
    } finally {
      setIsAdding(false)
    }
  }

  const invalidatePixEz = async () => {
    await Promise.all([
      queryClient.invalidateQueries({queryKey: ["pixez"]}),
      queryClient.invalidateQueries({queryKey: ["admin", "task-executions"]}),
    ])
  }

  const handleSync = async (account?: PixezAccount) => {
    const id = account?.pixiv_user_id ?? "all"
    try {
      setSyncingID(id)
      const payload = account ? JSON.stringify({pixiv_user_id: account.pixiv_user_id}) : ""
      const [illustTaskID, novelTaskID] = await Promise.all([
        AdminService.dispatchTask({
          task_type: "pixez_export_bookmark_illusts",
          payload,
        }),
        AdminService.dispatchTask({
          task_type: "pixez_export_bookmark_novels",
          payload,
        }),
      ])
      toast.success("收藏同步任务已入队", {
        description: `${illustTaskID} / ${novelTaskID}`,
      })
      await invalidatePixEz()
    } catch (error) {
      toast.error("同步收藏失败", {
        description: error instanceof Error ? error.message : "未知错误",
      })
    } finally {
      setSyncingID(null)
    }
  }


  const handleDelete = async (account: PixezAccount) => {
    try {
      setDeletingID(account.pixiv_user_id)
      await PixezService.deleteAccount(account.pixiv_user_id)
      toast.success("Pixiv 账号已断开")
      await invalidatePixEz()
    } catch (error) {
      toast.error("断开账号失败", {
        description: error instanceof Error ? error.message : "未知错误",
      })
    } finally {
      setDeletingID(null)
    }
  }

  return (
    <div className="flex w-full flex-col gap-5 py-6">
      <div className="flex items-center gap-3 border-b border-border pb-3">
        <div className="flex size-10 items-center justify-center rounded-md border bg-card">
          <UsersRound className="text-muted-foreground" />
        </div>
        <div className="flex flex-col gap-1">
          <h1 className="text-2xl font-semibold tracking-tight">账号管理</h1>
          <p className="text-xs text-muted-foreground">Pixiv 账号授权与数据导入同步</p>
        </div>
      </div>

      <div className="flex flex-col gap-4">
        <div className="flex flex-col gap-2 sm:flex-row sm:items-center sm:justify-between">
          <div className="text-sm font-medium">账号同步管理</div>
          <div className="flex flex-wrap gap-2">
            <Button variant="outline" size="sm" onClick={() => accountsQuery.refetch()} disabled={accountsQuery.isFetching}>
              {accountsQuery.isFetching ? <Spinner /> : <RefreshCw />}
              刷新
            </Button>
            <Button size="sm" onClick={() => handleSync()} disabled={syncingID !== null || accounts.length === 0}>
              {syncingID === "all" ? <Spinner /> : <Zap />}
              同步全部
            </Button>
            <Button variant="outline" size="sm" onClick={() => setIsAddOpen(true)}>
              <Plus />
              添加账号
            </Button>
          </div>
        </div>

        {accountsQuery.error ? (
          <div className="rounded-md border p-3">
            <ErrorInline error={accountsQuery.error} onRetry={() => accountsQuery.refetch()} />
          </div>
        ) : accountsQuery.isLoading ? (
          <LoadingStateWithBorder description="加载 Pixiv 账号中..." />
        ) : accounts.length === 0 ? (
          <EmptyStateWithBorder description="暂无 Pixiv 账号" />
        ) : (
          <div className="grid gap-3 lg:grid-cols-2 2xl:grid-cols-3">
            {accounts.map((account) => (
              <AccountCard
                key={account.pixiv_user_id}
                account={account}
                syncing={syncingID === account.pixiv_user_id}
                deleting={deletingID === account.pixiv_user_id}
                onSync={handleSync}
                onDelete={handleDelete}
              />
            ))}
          </div>
        )}
      </div>

      <Dialog open={isAddOpen} onOpenChange={(open) => {
        if (!isAdding) {
          setIsAddOpen(open)
          if (!open) {
            setRefreshToken("")
            setCallbackUrl("")
            setCodeVerifier("")
          }
        }
      }}>
        <DialogContent className="sm:max-w-[425px]">
          <DialogHeader>
            <DialogTitle>添加 Pixiv 账号</DialogTitle>
            <DialogDescription>
              您可以选择登录 Pixiv 自动获取凭证，或者手动输入刷新令牌。
            </DialogDescription>
          </DialogHeader>

          <Tabs defaultValue="login" className="w-full">
            <TabsList className="grid w-full grid-cols-2">
              <TabsTrigger value="login">登录 Pixiv 获取</TabsTrigger>
              <TabsTrigger value="manual">手动输入令牌</TabsTrigger>
            </TabsList>

            <TabsContent value="login" className="space-y-4 py-4">
              <div className="space-y-3">
                <div className="text-xs text-muted-foreground leading-normal space-y-2">
                  <p>
                    由于桌面浏览器无法直接处理手机客户端的 <code className="bg-muted px-1 py-0.5 rounded text-foreground">pixiv://</code> 跳转协议，地址栏通常会回退或卡在 <code className="bg-muted px-1 py-0.5 rounded text-foreground">post-redirect</code> 页面。请按照以下步骤获取登录授权码：
                  </p>
                  <ol className="list-decimal list-inside space-y-1 pl-1">
                    <li>在浏览器中按下 <kbd className="bg-muted px-1 py-0.5 rounded">F12</kbd> (或右键检查) 打开开发者工具。</li>
                    <li>切换到 <strong>“网络 (Network)”</strong> 标签页，并勾选 <strong>“保留日志 (Preserve log)”</strong>。</li>
                    <li>点击下方按钮打开登录页面，输入账号密码并完成登录。</li>
                    <li>登录成功后，在网络的请求列表中搜索过滤关键词 <code className="bg-muted px-1 py-0.5 rounded text-foreground">callback</code>，点击该请求并复制其 URL 中的 <code className="bg-muted px-1 py-0.5 rounded text-foreground">code</code> 参数值（或复制响应头中 Location 对应的整条链接）。</li>
                  </ol>
                </div>
                <Button
                  type="button"
                  variant="outline"
                  className="w-full"
                  onClick={handleOpenLogin}
                  disabled={isFetchingUrl || isAdding}
                >
                  {isFetchingUrl ? <Spinner className="mr-2" /> : <Chrome className="mr-2 size-4" />}
                  {isFetchingUrl ? "获取登录链接中..." : "打开 Pixiv 登录页面"}
                </Button>
              </div>

              <form onSubmit={handleCallbackSubmit} className="space-y-4">
                <div className="space-y-2">
                  <label htmlFor="callback-url" className="text-xs font-medium text-foreground">
                    回调 URL 或 Code
                  </label>
                  <Input
                    id="callback-url"
                    placeholder="粘贴以 pixiv://... 开头的地址"
                    value={callbackUrl}
                    onChange={(e) => setCallbackUrl(e.target.value)}
                    disabled={isAdding || !codeVerifier}
                    required
                  />
                  {!codeVerifier && (
                    <p className="text-[10px] text-destructive">请先点击上方按钮登录 Pixiv</p>
                  )}
                </div>
                <DialogFooter className="pt-2">
                  <Button type="button" variant="outline" size="sm" onClick={() => setIsAddOpen(false)} disabled={isAdding}>
                    取消
                  </Button>
                  <Button type="submit" size="sm" disabled={isAdding || !codeVerifier}>
                    {isAdding && <Spinner className="mr-2" />}
                    {isAdding ? "添加中..." : "确认"}
                  </Button>
                </DialogFooter>
              </form>
            </TabsContent>

            <TabsContent value="manual" className="space-y-4 py-4">
              <form onSubmit={handleAddAccount} className="space-y-4">
                <div className="space-y-2">
                  <label htmlFor="refresh-token" className="text-xs font-medium text-foreground">
                    刷新令牌 (Refresh Token)
                  </label>
                  <Input
                    id="refresh-token"
                    placeholder="输入 Pixiv 刷新令牌"
                    value={refreshToken}
                    onChange={(e) => setRefreshToken(e.target.value)}
                    disabled={isAdding}
                    required
                  />
                </div>
                <DialogFooter className="pt-2">
                  <Button type="button" variant="outline" size="sm" onClick={() => setIsAddOpen(false)} disabled={isAdding}>
                    取消
                  </Button>
                  <Button type="submit" size="sm" disabled={isAdding}>
                    {isAdding && <Spinner className="mr-2" />}
                    {isAdding ? "添加中..." : "确认"}
                  </Button>
                </DialogFooter>
              </form>
            </TabsContent>
          </Tabs>
        </DialogContent>
      </Dialog>
    </div>
  )
}
