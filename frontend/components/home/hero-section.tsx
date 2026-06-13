import * as React from "react";
import Link from "next/link";
import {motion} from "motion/react";
import type {LucideIcon} from "lucide-react";
import {
  ArrowRight,
  BookOpen,
  Cloud,
  GalleryVerticalEnd,
  HardDriveDownload,
  KeyRound,
  PackageCheck,
  ServerCog,
  ShieldCheck,
  Smartphone,
} from "lucide-react";
import {cn} from "@/lib/utils";
import {Button} from "@/components/ui/button";

export interface HeroSectionProps {
  className?: string;
}

const capabilities: Array<{
  icon: LucideIcon;
  label: string;
  description: string;
}> = [
  {
    icon: KeyRound,
    label: "一键恢复登录",
    description: "换个设备登录 Pixiv？不用再翻找密码。Pixez Cloud 帮你保管登录凭证，新设备上点一下就能恢复账号状态。",
  },
  {
    icon: GalleryVerticalEnd,
    label: "插画小说镜像",
    description: "辛苦收藏的插画/小说突然就没了？云端定期为你镜像收藏的作品，失效了直接换源。",
  },
  {
    icon: PackageCheck,
    label: "收藏不怕丢",
    description: "辛苦攒下的几千个收藏，万一丢失怎么办？云端定期备份你的收藏列表，还能帮你追踪哪些作品已经失效。",
  },
  {
    icon: Smartphone,
    label: "多设备随心切换",
    description: "手机、平板、多台手机——所有设备的浏览记录、屏蔽设置、搜索历史，统统保持一致。",
  },
];

const setupHighlights = [
  {
    title: "Docker 一键部署",
    detail: "5 分钟拥有专属服务。",
  },
  {
    title: "完全开源透明",
    detail: "基于 PixEz 二次开发, 完全开源",
  },
  {
    title: "数据由你掌控",
    detail: "支持本地存储或云对象存储。",
  },
];

const cloudSignals = ["Pixiv 账号凭证", "浏览记录", "屏蔽名单", "搜索历史", "收藏列表"];

/**
 * Hero Section - 首页 Hero 展示
 */
export const HeroSection = React.memo(function HeroSection({ className }: HeroSectionProps) {
  return (
    <section className={cn("w-full", className)}>
      <motion.div
        initial={{ opacity: 0, y: 28 }}
        whileInView={{ opacity: 1, y: 0 }}
        viewport={{ once: true }}
        transition={{
          delay: 0.15,
          duration: 0.7,
          ease: "easeInOut",
        }}
        className="relative z-10 flex min-h-[88vh] w-full items-center px-6 py-24 lg:py-20"
      >
        <div className="container mx-auto grid max-w-7xl items-center gap-10 lg:grid-cols-[1.02fr_0.98fr] lg:gap-16">
          <div className="max-w-3xl">
            <motion.h1
              initial={{ opacity: 0, y: 20 }}
              whileInView={{ opacity: 1, y: 0 }}
              viewport={{ once: true }}
              transition={{ duration: 0.6, delay: 0.1, ease: [0.16, 1, 0.3, 1] }}
              className="mb-6 text-4xl font-semibold leading-[1.08] text-foreground md:text-5xl lg:text-6xl"
            >
              Pixiv Sync
              <br />
              <span className="text-primary">Pixiv 的同步备份服务</span>
            </motion.h1>

            <motion.p
              initial={{ opacity: 0, y: 20 }}
              whileInView={{ opacity: 1, y: 0 }}
              viewport={{ once: true }}
              transition={{ duration: 0.6, delay: 0.2, ease: [0.16, 1, 0.3, 1] }}
              className="mb-8 max-w-2xl text-sm leading-7 text-muted-foreground md:text-base"
            >
              Pixiv Sync 是专为 Pixiv / PixEz 用户打造的云端备份服务。无论你拥有几台设备，
              它都能让你的 Pixiv 账号凭证、浏览记录、屏蔽名单等关键数据无缝同步。
            </motion.p>

            <motion.div
              initial={{ opacity: 0, y: 20 }}
              whileInView={{ opacity: 1, y: 0 }}
              viewport={{ once: true }}
              transition={{ duration: 0.6, delay: 0.3, ease: [0.16, 1, 0.3, 1] }}
              className="flex flex-col gap-3 sm:flex-row"
            >
              <Link href="/home" className="w-full sm:w-auto">
                <Button
                  size="lg"
                  className="w-full font-medium transition-all active:scale-95"
                >
                  进入仪表盘
                  <ArrowRight className="size-4" />
                </Button>
              </Link>

              <Link href="/docs/how-to-use" className="w-full sm:w-auto">
                <Button
                  variant="outline"
                  size="lg"
                  className="w-full font-medium active:scale-95"
                >
                  <BookOpen className="size-4" />
                  查看文档
                </Button>
              </Link>
            </motion.div>

            <motion.div
              initial={{ opacity: 0 }}
              whileInView={{ opacity: 1 }}
              viewport={{ once: true }}
              transition={{ duration: 0.8, delay: 0.5 }}
              className="mt-12 grid gap-3 border-t border-border pt-6 md:grid-cols-2"
            >
              {capabilities.map((item) => {
                const Icon = item.icon;

                return (
                  <div key={item.label} className="flex gap-3 rounded-lg border bg-card/60 p-3">
                    <div className="flex size-9 shrink-0 items-center justify-center rounded-md border bg-background">
                      <Icon className="size-4 text-primary" />
                    </div>
                    <div className="min-w-0">
                      <div className="text-sm font-medium text-foreground">{item.label}</div>
                      <p className="mt-1 text-xs leading-5 text-muted-foreground">{item.description}</p>
                    </div>
                  </div>
                );
              })}
            </motion.div>
          </div>

          <div className="relative w-full">
            <motion.div
              initial={{ opacity: 0, x: 36 }}
              whileInView={{ opacity: 1, x: 0 }}
              viewport={{ once: true }}
              transition={{ duration: 1, delay: 0.2, ease: "easeOut" }}
              className="w-full rounded-lg border bg-card/70 p-4 shadow-xl backdrop-blur-xl"
            >
              <div className="flex items-start justify-between gap-4 border-b pb-4">
                <div className="flex items-center gap-3">
                  <div className="flex size-10 items-center justify-center rounded-md bg-primary text-primary-foreground">
                    <Cloud className="size-5" />
                  </div>
                  <div>
                    <div className="text-sm font-semibold">Pixez Sync</div>
                    <div className="text-xs text-muted-foreground">专属 PixEz 云端数据伴侣</div>
                  </div>
                </div>
                <div className="rounded-md border px-2 py-1 text-xs text-muted-foreground">private cloud</div>
              </div>

              <div className="mt-4 space-y-3">
                <div className="rounded-lg border bg-background/70 p-4">
                  <div className="mb-3 text-sm font-medium text-foreground">它能同步什么</div>
                  <div className="flex flex-wrap gap-2">
                    {cloudSignals.map((signal) => (
                      <span key={signal} className="rounded-md bg-muted px-2 py-1 text-xs text-muted-foreground">
                        {signal}
                      </span>
                    ))}
                  </div>
                </div>

                {setupHighlights.map((group, index) => (
                  <div key={group.title} className="grid grid-cols-[auto_1fr] gap-3 rounded-lg border bg-background/70 p-3">
                    <div className="flex size-8 items-center justify-center rounded-md border bg-muted text-xs font-semibold text-muted-foreground">
                      {index + 1}
                    </div>
                    <div className="min-w-0">
                      <span className="text-sm font-medium text-foreground">{group.title}</span>
                      <p className="mt-1 text-xs leading-5 text-muted-foreground">{group.detail}</p>
                    </div>
                  </div>
                ))}
              </div>

              <div className="mt-4 grid gap-3 sm:grid-cols-2">
                <div className="rounded-lg border bg-background/70 p-3">
                  <div className="mb-3 flex items-center gap-2 text-sm font-medium">
                    <HardDriveDownload className="size-4 text-primary" />
                    本地或对象存储
                  </div>
                  <p className="text-xs leading-5 text-muted-foreground">
                    镜像文件可以保存在自己的服务器，也可以接入 S3 兼容云存储。
                  </p>
                </div>
                <div className="rounded-lg border bg-background/70 p-3">
                  <div className="mb-3 flex items-center gap-2 text-sm font-medium">
                    <ServerCog className="size-4 text-primary" />
                    管理端可视化
                  </div>
                  <div className="space-y-2 text-xs text-muted-foreground">
                    <div className="flex items-center gap-2">
                      <ShieldCheck className="size-3.5 text-primary" />
                      AccessToken 鉴权
                    </div>
                    <div className="flex items-center gap-2">
                      <PackageCheck className="size-3.5 text-primary" />
                      任务与收藏备份可观测
                    </div>
                  </div>
                </div>
              </div>
            </motion.div>
          </div>
        </div>
      </motion.div>
    </section>
  );
});
