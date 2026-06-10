"use client"

import {useState} from "react"
import {RefreshCcw, Trash2, Zap} from "lucide-react"

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
import {Avatar, AvatarFallback, AvatarImage} from "@/components/ui/avatar"
import {Badge} from "@/components/ui/badge"
import {Button} from "@/components/ui/button"
import {Card, CardContent, CardFooter, CardHeader, CardTitle} from "@/components/ui/card"
import {Spinner} from "@/components/ui/spinner"
import type {PixezAccount} from "@/lib/services"

import {formatPixEzDateTime} from "./pixez-format"

export function AccountCard({
  account,
  syncing,
  refreshing,
  deleting,
  onSync,
  onRefresh,
  onDelete,
}: {
  account: PixezAccount
  syncing?: boolean
  refreshing?: boolean
  deleting?: boolean
  onSync: (account: PixezAccount) => Promise<void>
  onRefresh: (account: PixezAccount) => Promise<void>
  onDelete: (account: PixezAccount) => Promise<void>
}) {
  const [deleteOpen, setDeleteOpen] = useState(false)

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
          <Button variant="outline" size="sm" onClick={() => onRefresh(account)} disabled={syncing || refreshing || deleting}>
            {refreshing ? <Spinner /> : <RefreshCcw />}
            刷新凭证
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
    </>
  )
}
