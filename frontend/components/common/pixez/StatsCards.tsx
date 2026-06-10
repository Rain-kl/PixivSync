import type {ComponentType} from "react"
import {Activity, BookOpenText, ImageIcon, UsersRound} from "lucide-react"

import {Card, CardContent, CardDescription, CardHeader, CardTitle} from "@/components/ui/card"
import {Progress} from "@/components/ui/progress"
import {Skeleton} from "@/components/ui/skeleton"
import type {PixezDashboard, PixezMirrorProgress} from "@/lib/services"

import {formatPixEzNumber, formatPixEzPercent} from "./pixez-format"

function ProgressCard({
  title,
  description,
  progress,
  icon: Icon,
}: {
  title: string
  description: string
  progress: PixezMirrorProgress
  icon: ComponentType<{className?: string}>
}) {
  return (
    <Card className="rounded-lg">
      <CardHeader className="gap-2">
        <div className="flex items-center justify-between gap-3">
          <div className="flex flex-col gap-1">
            <CardDescription>{description}</CardDescription>
            <CardTitle className="text-xl">{title}</CardTitle>
          </div>
          <div className="flex size-9 items-center justify-center rounded-md border bg-background">
            <Icon className="text-muted-foreground" />
          </div>
        </div>
      </CardHeader>
      <CardContent className="flex flex-col gap-3">
        <div className="flex items-baseline justify-between gap-3">
          <div className="text-2xl font-semibold tabular-nums">
            {formatPixEzNumber(progress.succeeded)}
            <span className="text-sm font-normal text-muted-foreground"> / {formatPixEzNumber(progress.total)}</span>
          </div>
          <div className="text-sm font-medium tabular-nums">{formatPixEzPercent(progress.percent)}</div>
        </div>
        <Progress value={progress.percent} />
        <div className="grid grid-cols-3 gap-2 text-[11px] text-muted-foreground">
          <span>下载中 {formatPixEzNumber(progress.processing)}</span>
          <span>失败 {formatPixEzNumber(progress.failed)}</span>
          <span>未入队 {formatPixEzNumber(progress.not_queued)}</span>
        </div>
      </CardContent>
    </Card>
  )
}

function StatsSkeleton() {
  return (
    <div className="grid gap-3 sm:grid-cols-2 xl:grid-cols-4">
      {Array.from({length: 4}).map((_, index) => (
        <Card key={index} className="rounded-lg">
          <CardHeader>
            <Skeleton className="h-4 w-24" />
            <Skeleton className="h-7 w-32" />
          </CardHeader>
          <CardContent className="flex flex-col gap-3">
            <Skeleton className="h-8 w-full" />
            <Skeleton className="h-2 w-full" />
          </CardContent>
        </Card>
      ))}
    </div>
  )
}

export function StatsCards({dashboard, loading}: {dashboard?: PixezDashboard; loading?: boolean}) {
  if (loading || !dashboard) {
    return <StatsSkeleton />
  }

  return (
    <div className="grid gap-3 sm:grid-cols-2 xl:grid-cols-4">
      <Card className="rounded-lg">
        <CardHeader className="gap-2">
          <div className="flex items-center justify-between gap-3">
            <div className="flex flex-col gap-1">
              <CardDescription>同步账号数</CardDescription>
              <CardTitle className="text-xl">Pixiv 账号</CardTitle>
            </div>
            <div className="flex size-9 items-center justify-center rounded-md border bg-background">
              <UsersRound className="text-muted-foreground" />
            </div>
          </div>
        </CardHeader>
        <CardContent className="flex flex-col gap-3">
          <div className="text-3xl font-semibold tabular-nums">{formatPixEzNumber(dashboard.accounts)}</div>
          <div className="text-xs text-muted-foreground">已绑定并维护的账号</div>
        </CardContent>
      </Card>

      <ProgressCard
        title="插画镜像"
        description="插画镜像进度"
        progress={dashboard.illusts}
        icon={ImageIcon}
      />
      <ProgressCard
        title="小说镜像"
        description="小说镜像进度"
        progress={dashboard.novels}
        icon={BookOpenText}
      />

      <Card className="rounded-lg">
        <CardHeader className="gap-2">
          <div className="flex items-center justify-between gap-3">
            <div className="flex flex-col gap-1">
              <CardDescription>活跃队列任务</CardDescription>
              <CardTitle className="text-xl">Asynq 队列</CardTitle>
            </div>
            <div className="flex size-9 items-center justify-center rounded-md border bg-background">
              <Activity className="text-muted-foreground" />
            </div>
          </div>
        </CardHeader>
        <CardContent className="flex flex-col gap-3">
          <div className="text-3xl font-semibold tabular-nums">
            {formatPixEzNumber(dashboard.queue.running)}
            <span className="text-sm font-normal text-muted-foreground"> 运行中 / </span>
            {formatPixEzNumber(dashboard.queue.queued)}
            <span className="text-sm font-normal text-muted-foreground"> 排队中</span>
          </div>
          <div className="text-xs text-muted-foreground">PixEz 相关任务执行记录</div>
        </CardContent>
      </Card>
    </div>
  )
}
