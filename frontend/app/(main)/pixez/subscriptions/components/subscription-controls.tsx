"use client"

import {useState} from "react"
import {useMutation, useQueryClient} from "@tanstack/react-query"
import {RefreshCw} from "lucide-react"
import {toast} from "sonner"
import {ArtistService} from "@/lib/services"
import type {ArtistSubscription} from "@/lib/services/pixez/artists.service"
import type {PixezMirrorTarget} from "@/lib/services"
import {Button} from "@/components/ui/button"
import {Switch} from "@/components/ui/switch"
import {Spinner} from "@/components/ui/spinner"
import {AlertDialog, AlertDialogAction, AlertDialogCancel, AlertDialogContent, AlertDialogDescription, AlertDialogFooter, AlertDialogHeader, AlertDialogTitle} from "@/components/ui/alert-dialog"

export function SubscriptionControls({id, target, subscription}: {
  id: string; target: PixezMirrorTarget; subscription?: ArtistSubscription
}) {
  const queryClient = useQueryClient()
  const [confirmCancel, setConfirmCancel] = useState(false)
  const invalidate = () => queryClient.invalidateQueries({queryKey: ["artists"]})
  const onError = (error: Error) => toast.error(error.message)
  const toggle = useMutation({
    mutationFn: (enabled: boolean) => ArtistService.setSubscription(id, target, enabled),
    onSuccess: (_, enabled) => toast.success(enabled ? "订阅已开启，开始同步历史作品" : "已取消订阅，已有备份保留"),
    onError, onSettled: invalidate,
  })
  const sync = useMutation({
    mutationFn: () => ArtistService.sync(id, target),
    onSuccess: () => toast.success("同步已安排，请查看扫描进度"),
    onError, onSettled: invalidate,
  })
  const busy = toggle.isPending || sync.isPending
  const enabled = subscription?.enabled ?? false
  const name = target === "novel" ? "小说" : "插画"
  return (
    <div className="flex flex-wrap items-center gap-3">
      <Switch id={`subscription-${id}-${target}`} checked={enabled} disabled={busy}
        onCheckedChange={value => value ? toggle.mutate(true) : setConfirmCancel(true)} />
      <label htmlFor={`subscription-${id}-${target}`} className="text-sm">{name}订阅</label>
      {enabled && <Button variant="outline" size="sm" disabled={busy || !!subscription?.active_run_id} onClick={() => sync.mutate()}>
        {busy ? <Spinner /> : <RefreshCw data-icon="inline-start" />}
        {subscription?.active_run_id ? "正在扫描" : "立即同步"}
      </Button>}
      <AlertDialog open={confirmCancel} onOpenChange={setConfirmCancel}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>取消{name}订阅？</AlertDialogTitle>
            <AlertDialogDescription>这会停止全站对该作者{name}的后续自动同步。已有备份继续保留，另一种作品的订阅不受影响。</AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>继续订阅</AlertDialogCancel>
            <AlertDialogAction onClick={() => toggle.mutate(false)}>取消订阅并保留备份</AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </div>
  )
}
