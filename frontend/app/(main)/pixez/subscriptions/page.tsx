"use client"

import {useState} from "react"
import {Bell} from "lucide-react"
import {Tabs, TabsContent, TabsList, TabsTrigger} from "@/components/ui/tabs"
import {SubscriptionList} from "./components/subscription-list"

export default function SubscriptionsPage() {
  const [tab, setTab] = useState("novel")
  return (
    <div className="flex w-full flex-col gap-5 py-6">
      <div className="flex flex-col gap-2">
        <div className="flex items-center gap-2">
          <Bell className="size-5 text-primary" />
          <h1 className="text-2xl font-semibold tracking-tight">订阅管理</h1>
        </div>
        <p className="text-sm text-muted-foreground">从已有镜像发现作者，订阅后立即备份历史作品，此后每天检查新作品。全站共享订阅与备份。</p>
      </div>
      <Tabs value={tab} onValueChange={setTab}>
        <TabsList>
          <TabsTrigger value="novel">小说订阅</TabsTrigger>
          <TabsTrigger value="illust">插画订阅</TabsTrigger>
        </TabsList>
        <TabsContent value="novel"><SubscriptionList target="novel" /></TabsContent>
        <TabsContent value="illust"><SubscriptionList target="illust" /></TabsContent>
      </Tabs>
    </div>
  )
}
