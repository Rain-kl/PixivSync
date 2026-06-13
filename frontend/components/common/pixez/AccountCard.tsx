"use client"

import {useState} from "react"
import {Info, Trash2, Zap} from "lucide-react"

import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from "@/components/ui/alert-dialog"
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog"
import {Avatar, AvatarFallback, AvatarImage} from "@/components/ui/avatar"
import {Badge} from "@/components/ui/badge"
import {Button} from "@/components/ui/button"
import {Card, CardContent, CardFooter, CardHeader, CardTitle} from "@/components/ui/card"
import {Spinner} from "@/components/ui/spinner"
import {PixezService} from "@/lib/services"
import type {PixezAccount, PixivProfileResponse} from "@/lib/services"
import {toast} from "sonner"

import {formatPixEzDateTime} from "./pixez-format"

export function AccountCard({
  account,
  syncing,
  refreshing,
  deleting,
  onSync,
  onDelete,
}: {
  account: PixezAccount
  syncing?: boolean
  refreshing?: boolean
  deleting?: boolean
  onSync: (account: PixezAccount) => Promise<void>
  onDelete: (account: PixezAccount) => Promise<void>
}) {
  const [deleteOpen, setDeleteOpen] = useState(false)
  const [infoOpen, setInfoOpen] = useState(false)
  const [profile, setProfile] = useState<PixivProfileResponse | null>(null)
  const [loadingProfile, setLoadingProfile] = useState(false)

  const handleViewInfo = async () => {
    setInfoOpen(true)
    setLoadingProfile(true)
    try {
      const data = await PixezService.getUserProfile(account.pixiv_user_id)
      setProfile(data)
    } catch (error) {
      toast.error("获取 Pixiv 个人信息失败", {
        description: error instanceof Error ? error.message : "未知错误",
      })
    } finally {
      setLoadingProfile(false)
    }
  }

  const tokenStatus = refreshing ? "自动刷新中" : "已保存"

  return (
    <>
      <Card className="rounded-lg">
        <CardHeader>
          <div className="flex items-start gap-3">
            <Avatar className="size-12 rounded-md">
              <AvatarImage src={account.user_image} alt={account.name} />
              <AvatarFallback className="rounded-md">{account.name?.slice(0, 1) || "P"}</AvatarFallback>
            </Avatar>
            <div className="min-w-0 flex-1">
              <CardTitle className="truncate text-base">{account.name || account.account || account.pixiv_user_id}</CardTitle>
              <div className="mt-1 flex flex-wrap items-center gap-2">
                <Badge variant="outline" className="font-mono">{account.pixiv_user_id}</Badge>
                {account.is_premium === 1 && <Badge variant="secondary">Premium</Badge>}
              </div>
            </div>
          </div>
        </CardHeader>
        <CardContent className="grid gap-3 text-sm">
          <div className="flex items-center justify-between gap-3">
            <span className="text-muted-foreground">Access Token</span>
            <Badge variant={refreshing ? "outline" : "secondary"}>{tokenStatus}</Badge>
          </div>
          <div className="flex items-center justify-between gap-3">
            <span className="text-muted-foreground">最近维护</span>
            <span className="font-mono text-xs">{formatPixEzDateTime(account.updated_at)}</span>
          </div>
          <div className="flex items-center justify-between gap-3">
            <span className="text-muted-foreground">账号标识</span>
            <span className="truncate text-xs">{account.account || account.mail_address || "-"}</span>
          </div>
        </CardContent>
        <CardFooter className="flex flex-wrap gap-2">
          <Button size="sm" onClick={() => onSync(account)} disabled={syncing || refreshing || deleting}>
            {syncing ? <Spinner /> : <Zap />}
            同步收藏
          </Button>
          <Button variant="outline" size="sm" onClick={handleViewInfo} disabled={syncing || refreshing || deleting}>
            <Info />
            查看信息
          </Button>
          <Button variant="outline" size="sm" onClick={() => setDeleteOpen(true)} disabled={syncing || refreshing || deleting}>
            <Trash2 />
            断开
          </Button>
        </CardFooter>
      </Card>

      <AlertDialog open={deleteOpen} onOpenChange={(open) => !deleting && setDeleteOpen(open)}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>断开 Pixiv 账号</AlertDialogTitle>
            <AlertDialogDescription>
              将删除该账号凭证与同步数据，已有镜像文件不会在此操作中批量清理。
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel disabled={deleting}>取消</AlertDialogCancel>
            <AlertDialogAction
              onClick={async () => {
                await onDelete(account)
                setDeleteOpen(false)
              }}
              disabled={deleting}
            >
              {deleting && <Spinner />}
              确认断开
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>

      <Dialog open={infoOpen} onOpenChange={setInfoOpen}>
        <DialogContent className="sm:max-w-[450px]">
          <DialogHeader className="flex flex-col items-center border-b border-dashed pb-4">
            <Avatar className="size-20 rounded-full border-2 border-primary/20 shadow-md">
              <AvatarImage src={profile?.user?.profile_image_urls?.medium || account.user_image} alt={profile?.user?.name || account.name} />
              <AvatarFallback className="text-xl">{(profile?.user?.name || account.name)?.slice(0, 1) || "P"}</AvatarFallback>
            </Avatar>
            <DialogTitle className="mt-3 text-lg font-bold text-center">
              {profile?.user?.name || account.name || "未指定昵称"}
            </DialogTitle>
            <div className="mt-1 flex items-center gap-2">
              <Badge variant="outline" className="font-mono text-xs">
                ID: {account.pixiv_user_id}
              </Badge>
              {profile ? (
                <Badge className={profile.profile?.is_premium ? "bg-amber-500 hover:bg-amber-600 text-white border-0 text-xs" : "text-xs"} variant={profile.profile?.is_premium ? "default" : "secondary"}>
                  {profile.profile?.is_premium ? "Premium" : "Regular"}
                </Badge>
              ) : account.is_premium === 1 ? (
                <Badge className="bg-amber-500 hover:bg-amber-600 text-white border-0 text-xs">
                  Premium
                </Badge>
              ) : (
                <Badge variant="secondary" className="text-xs">
                  Regular
                </Badge>
              )}
            </div>
          </DialogHeader>

          {loadingProfile ? (
            <div className="flex h-44 items-center justify-center">
              <Spinner className="size-8" />
            </div>
          ) : profile ? (
            <div className="grid gap-3 py-4 text-xs">
              {profile.user?.comment && (
                <div className="flex flex-col gap-1 border-b border-border/40 pb-2">
                  <span className="text-muted-foreground font-medium">个人简介</span>
                  <p className="text-foreground leading-relaxed break-all font-normal whitespace-pre-wrap">{profile.user.comment}</p>
                </div>
              )}
              <div className="flex items-center justify-between border-b border-border/40 pb-2">
                <span className="text-muted-foreground font-medium">Pixiv 账号名 (Account)</span>
                <span className="font-mono text-foreground font-semibold">{profile.user?.account || "-"}</span>
              </div>
              <div className="flex items-center justify-between border-b border-border/40 pb-2">
                <span className="text-muted-foreground font-medium">性别 / 地区</span>
                <span className="text-foreground">
                  {[profile.profile?.gender, profile.profile?.region].filter(Boolean).join(" / ") || "-"}
                </span>
              </div>
              <div className="flex items-center justify-between border-b border-border/40 pb-2">
                <span className="text-muted-foreground font-medium">作品数量</span>
                <span className="text-foreground">
                  插画 {profile.profile?.total_illusts ?? 0} | 小说 {profile.profile?.total_novels ?? 0}
                </span>
              </div>
              <div className="flex items-center justify-between border-b border-border/40 pb-2">
                <span className="text-muted-foreground font-medium">公开收藏插画数</span>
                <span className="font-mono text-foreground">{profile.profile?.total_illust_bookmarks_public ?? 0}</span>
              </div>
              <div className="flex items-center justify-between border-b border-border/40 pb-2">
                <span className="text-muted-foreground font-medium">关注画师数</span>
                <span className="font-mono text-foreground">{profile.profile?.total_follow_users ?? 0}</span>
              </div>
              {profile.profile?.twitter_account && (
                <div className="flex items-center justify-between border-b border-border/40 pb-2">
                  <span className="text-muted-foreground font-medium">Twitter</span>
                  <span className="font-mono text-foreground">@{profile.profile.twitter_account}</span>
                </div>
              )}
              <div className="flex items-center justify-between pb-1">
                <span className="text-muted-foreground font-medium">凭证最近更新</span>
                <span className="font-mono text-muted-foreground">{formatPixEzDateTime(account.updated_at)}</span>
              </div>
            </div>
          ) : (
            <div className="grid gap-3 py-4 text-xs">
              <div className="flex items-center justify-between border-b border-border/40 pb-2">
                <span className="text-muted-foreground font-medium">Pixiv 账号名 (Account)</span>
                <span className="font-mono text-foreground font-semibold">{account.account || "-"}</span>
              </div>
              <div className="flex items-center justify-between border-b border-border/40 pb-2">
                <span className="text-muted-foreground font-medium">电子邮箱</span>
                <span className="font-mono text-foreground">{account.mail_address || "-"}</span>
              </div>
              <div className="flex items-center justify-between border-b border-border/40 pb-2">
                <span className="text-muted-foreground font-medium">限制等级 (R-18)</span>
                <Badge variant={account.x_restrict > 0 ? "destructive" : "secondary"}>
                  {account.x_restrict === 0 ? "全年龄 (Safe)" : account.x_restrict === 1 ? "限制级 (R-18)" : "限制级 (R-18G)"}
                </Badge>
              </div>
              <div className="flex items-center justify-between border-b border-border/40 pb-2">
                <span className="text-muted-foreground font-medium">绑定时间</span>
                <span className="font-mono text-muted-foreground">{formatPixEzDateTime(account.created_at)}</span>
              </div>
              <div className="flex items-center justify-between pb-1">
                <span className="text-muted-foreground font-medium">凭证最近刷新</span>
                <span className="font-mono text-muted-foreground">{formatPixEzDateTime(account.updated_at)}</span>
              </div>
            </div>
          )}
        </DialogContent>
      </Dialog>
    </>
  )
}
