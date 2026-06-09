"use client"

import * as React from "react"
import {useMemo, useState} from "react"
import {AnimatePresence, motion} from "motion/react"
import {Card, CardContent, CardDescription, CardHeader, CardTitle} from "@/components/ui/card"
import {Badge} from "@/components/ui/badge"
import {Button} from "@/components/ui/button"
import {Progress} from "@/components/ui/progress"
import {Tabs, TabsContent, TabsList, TabsTrigger} from "@/components/ui/tabs"
import {Activity, ArrowRight, Check, Code2, Coins, Copy, Gift, Info, Layers, Shield, User} from "lucide-react"
import {Area, AreaChart, ResponsiveContainer, Tooltip as RechartsTooltip, XAxis} from "recharts"
import {toast} from "sonner"

// ==========================================
// 1. 卡片类组件 (Cards)
// ==========================================

function FeatureCard({
  title,
  description,
  linkText,
  href
}: {
  title: string
  description: string
  linkText: string
  href?: string
}) {
  return (
    <Card className="bg-card border border-dashed shadow-none hover:shadow-sm transition-all duration-300">
      <CardHeader className="pb-2">
        <CardTitle className="text-sm font-semibold">{title}</CardTitle>
      </CardHeader>
      <CardContent className="space-y-3">
        <CardDescription className="text-xs text-muted-foreground leading-normal">
          {description}
        </CardDescription>
        <Button
          variant="link"
          className="p-0 h-auto text-xs text-indigo-500 font-medium hover:text-indigo-600 flex items-center gap-1"
          onClick={() => {
            if (href) {
              toast.success(`模拟跳转到: ${href}`)
            } else {
              toast.info("此按钮为静态演示，无实际链接")
            }
          }}
        >
          {linkText} <ArrowRight className="size-3" />
        </Button>
      </CardContent>
    </Card>
  )
}

function ReceiveBanner() {
  const [isVisible, setIsVisible] = useState(true)

  return (
    <AnimatePresence>
      {isVisible && (
        <motion.div
          initial={{ opacity: 0, height: 0 }}
          animate={{ opacity: 1, height: "auto" }}
          exit={{ opacity: 0, height: 0 }}
          transition={{ duration: 0.3 }}
          className="overflow-hidden mb-6"
        >
          <div className="bg-gradient-to-r from-indigo-500/10 via-purple-500/10 to-pink-500/10 border border-dashed border-indigo-500/20 rounded-xl p-6 relative">
            <div className="max-w-xl space-y-3">
              <h3 className="text-lg font-bold text-indigo-600 dark:text-indigo-400">获得积分收益</h3>
              <p className="text-xs text-muted-foreground leading-relaxed">
                通过无代码选项快速开始探索或使用与我们的 API 集成的可自定义积分服务。集成方式多样，满足各种场景需求。
              </p>
              <div className="flex gap-2.5">
                <Button
                  size="sm"
                  className="bg-indigo-600 hover:bg-indigo-700 text-xs font-semibold px-4 shadow-sm"
                  onClick={() => toast.success("正在进入积分探索模块...")}
                >
                  开始使用
                </Button>
                <Button
                  size="sm"
                  variant="ghost"
                  className="text-xs font-medium hover:bg-black/5 dark:hover:bg-white/5"
                  onClick={() => setIsVisible(false)}
                >
                  隐藏提示
                </Button>
              </div>
            </div>
          </div>
        </motion.div>
      )}
    </AnimatePresence>
  )
}

// ==========================================
// 2. 仪表盘组件 (Dashboards)
// ==========================================

const MOCK_CHART_DATA = [
  { date: "06-01", total: 12000, income: 400, expense: 200 },
  { date: "06-02", total: 13500, income: 1500, expense: 300 },
  { date: "06-03", total: 12800, income: 300, expense: 1000 },
  { date: "06-04", total: 15000, income: 2500, expense: 300 },
  { date: "06-05", total: 14200, income: 200, expense: 1000 },
  { date: "06-06", total: 16800, income: 3000, expense: 400 },
  { date: "06-07", total: 17500, income: 1200, expense: 500 },
]

function MockTrendChart() {
  return (
    <Card className="border border-dashed shadow-none rounded-lg p-5">
      <CardHeader className="p-0 pb-4">
        <CardTitle className="text-xs font-medium text-muted-foreground uppercase tracking-wider">积分变动趋势 (静态模拟)</CardTitle>
      </CardHeader>
      <div className="w-full h-[220px]">
        <ResponsiveContainer width="100%" height="100%">
          <AreaChart data={MOCK_CHART_DATA} margin={{ left: -10, right: 10, top: 10, bottom: 0 }}>
            <defs>
              <linearGradient id="colorTotal" x1="0" y1="0" x2="0" y2="1">
                <stop offset="5%" stopColor="rgb(99, 102, 241)" stopOpacity={0.2} />
                <stop offset="95%" stopColor="rgb(99, 102, 241)" stopOpacity={0} />
              </linearGradient>
            </defs>
            <XAxis dataKey="date" stroke="#888888" fontSize={10} tickLine={false} axisLine={false} />
            <RechartsTooltip
              contentStyle={{ background: "rgba(15, 23, 42, 0.9)", border: "none", borderRadius: "8px", color: "#fff" }}
              labelStyle={{ fontWeight: "bold", fontSize: "11px", marginBottom: "4px" }}
              itemStyle={{ fontSize: "11px", padding: "2px 0" }}
            />
            <Area
              name="总积分"
              dataKey="total"
              type="monotone"
              fill="url(#colorTotal)"
              stroke="rgb(99, 102, 241)"
              strokeWidth={2}
            />
          </AreaChart>
        </ResponsiveContainer>
      </div>
    </Card>
  )
}

function MockIncomeStats() {
  const stats = [
    { date: "06-07", amount: 1200, percentage: 80 },
    { date: "06-06", amount: 3000, percentage: 100 },
    { date: "06-05", amount: 200, percentage: 15 },
    { date: "06-04", amount: 2500, percentage: 90 },
    { date: "06-03", amount: 300, percentage: 20 },
    { date: "06-02", amount: 1500, percentage: 65 },
    { date: "06-01", amount: 400, percentage: 25 },
  ]

  return (
    <Card className="border border-dashed shadow-none rounded-lg p-5">
      <CardHeader className="p-0 pb-4 flex flex-row items-center justify-between">
        <div>
          <CardTitle className="text-xs font-medium text-muted-foreground uppercase tracking-wider">7天收入统计 (静态模拟)</CardTitle>
          <div className="text-lg font-bold mt-1">LDC 9,100.00</div>
        </div>
        <Button variant="outline" size="sm" className="h-6 text-[10px] px-2" onClick={() => toast.success("正在刷新统计数据...")}>
          刷新
        </Button>
      </CardHeader>
      <div className="space-y-2">
        {stats.map((s) => (
          <div key={s.date} className="space-y-1">
            <div className="flex justify-between text-[10px]">
              <span className="text-muted-foreground">{s.date}</span>
              <span className="text-emerald-500 font-semibold">+{s.amount.toFixed(2)}</span>
            </div>
            <Progress value={s.percentage} className="h-1.5 [&>[data-slot=progress-indicator]]:bg-emerald-500" />
          </div>
        ))}
      </div>
    </Card>
  )
}

function MockTopCustomers() {
  const customers = [
    { rank: 1, name: "dev_user", count: 42, amount: 28500 },
    { rank: 2, name: "linux_do_fan", count: 28, amount: 19800 },
    { rank: 3, name: "gorm_expert", count: 19, amount: 12400 },
    { rank: 4, name: "nextjs_coder", count: 12, amount: 8500 },
    { rank: 5, name: "redis_helper", count: 8, amount: 5000 },
  ]

  return (
    <Card className="border border-dashed shadow-none rounded-lg p-5">
      <CardHeader className="p-0 pb-4">
        <CardTitle className="text-xs font-medium text-muted-foreground uppercase tracking-wider">Top 客户统计 (静态模拟)</CardTitle>
      </CardHeader>
      <div className="space-y-3">
        {customers.map((c) => (
          <div key={c.rank} className="space-y-1">
            <div className="flex justify-between items-center text-xs">
              <div className="flex items-center gap-2">
                <span className="font-bold text-muted-foreground/80 w-4">#{c.rank}</span>
                <span className="font-medium">{c.name}</span>
              </div>
              <div className="font-semibold text-right">
                {c.amount.toLocaleString()} LDC
              </div>
            </div>
            <div className="flex items-center gap-2">
              <Progress value={(c.amount / 28500) * 100} className="h-1.5" />
              <span className="text-[10px] text-muted-foreground shrink-0 w-8 text-right">{c.count}单</span>
            </div>
          </div>
        ))}
      </div>
    </Card>
  )
}

// ==========================================
// 3. 数据表格组件 (Tables)
// ==========================================

interface MockOrder {
  id: string
  order_name: string
  amount: number
  type: 'payment' | 'transfer' | 'community' | 'red_envelope'
  status: 'success' | 'pending' | 'expired' | 'disputing' | 'refund'
  order_no: string
  created_at: string
}

const MOCK_ORDERS: MockOrder[] = [
  { id: "1", order_name: "API 接口充值积分", amount: 1000, type: "payment", status: "success", order_no: "ORD202606070001", created_at: "2026-06-07 14:20:15" },
  { id: "2", order_name: "退款原路退回", amount: 500, type: "payment", status: "refund", order_no: "ORD202606070002", created_at: "2026-06-07 12:05:44" },
  { id: "3", order_name: "拼手气红包分配", amount: 200, type: "red_envelope", status: "success", order_no: "ORD202606060012", created_at: "2026-06-06 18:30:00" },
  { id: "4", order_name: "在线流转纠纷处理", amount: 1500, type: "transfer", status: "disputing", order_no: "ORD202606050025", created_at: "2026-06-05 09:12:10" },
  { id: "5", order_name: "社区绑定自动转入", amount: 350, type: "community", status: "pending", order_no: "ORD202606040081", created_at: "2026-06-04 23:45:19" },
  { id: "6", order_name: "红包超时未领过期退回", amount: 100, type: "red_envelope", status: "expired", order_no: "ORD202606030009", created_at: "2026-06-03 15:00:00" },
]

function MockTransactionTable() {
  const getStatusBadge = (status: MockOrder["status"]) => {
    const map = {
      success: { text: "交易成功", class: "bg-green-500/10 text-green-600 border-green-500/20" },
      pending: { text: "等待付款", class: "bg-amber-500/10 text-amber-600 border-amber-500/20" },
      expired: { text: "订单过期", class: "bg-slate-500/10 text-slate-500 border-slate-500/20" },
      disputing: { text: "争议中", class: "bg-rose-500/10 text-rose-600 border-rose-500/20" },
      refund: { text: "交易退款", class: "bg-indigo-500/10 text-indigo-600 border-indigo-500/20" },
    }
    return (
      <Badge variant="outline" className={`text-[10px] px-1.5 py-0 font-medium ${map[status].class}`}>
        {map[status].text}
      </Badge>
    )
  }

  const getTypeBadge = (type: MockOrder["type"]) => {
    const map = {
      payment: { text: "消耗", class: "bg-orange-500/10 text-orange-600" },
      transfer: { text: "流转", class: "bg-blue-500/10 text-blue-600" },
      community: { text: "社区", class: "bg-purple-500/10 text-purple-600" },
      red_envelope: { text: "红包", class: "bg-red-500/10 text-red-600" },
    }
    return (
      <span className={`text-[10px] px-1.5 py-0.5 rounded font-medium ${map[type].class}`}>
        {map[type].text}
      </span>
    )
  }

  return (
    <Card className="border border-dashed shadow-none rounded-lg overflow-hidden">
      <div className="w-full overflow-x-auto">
        <table className="w-full caption-bottom text-xs">
          <thead>
            <tr className="border-b border-dashed bg-muted/20">
              <th className="h-10 px-4 text-left font-medium text-muted-foreground w-[180px]">名称</th>
              <th className="h-10 px-4 text-center font-medium text-muted-foreground w-[80px]">类型</th>
              <th className="h-10 px-4 text-center font-medium text-muted-foreground w-[90px]">额度</th>
              <th className="h-10 px-4 text-center font-medium text-muted-foreground w-[90px]">状态</th>
              <th className="h-10 px-4 text-left font-medium text-muted-foreground">流水编号</th>
              <th className="h-10 px-4 text-left font-medium text-muted-foreground w-[150px]">交易时间</th>
              <th className="h-10 px-4 text-center font-medium text-muted-foreground w-[80px]">操作</th>
            </tr>
          </thead>
          <tbody className="divide-y divide-dashed">
            {MOCK_ORDERS.map((order) => (
              <tr key={order.id} className="hover:bg-muted/30 transition-colors">
                <td className="p-4 font-medium">{order.order_name}</td>
                <td className="p-4 text-center">{getTypeBadge(order.type)}</td>
                <td className="p-4 text-center font-semibold text-slate-800 dark:text-slate-200">
                  {order.amount.toFixed(2)}
                </td>
                <td className="p-4 text-center">{getStatusBadge(order.status)}</td>
                <td className="p-4 font-mono text-muted-foreground">{order.order_no}</td>
                <td className="p-4 text-muted-foreground">{order.created_at}</td>
                <td className="p-4 text-center">
                  <Button
                    variant="link"
                    className="p-0 h-auto text-[11px] text-indigo-500"
                    onClick={() => toast.success(`查看详情: ${order.order_no}`)}
                  >
                    详情
                  </Button>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </Card>
  )
}

// ==========================================
// 4. Tab 导航组件 (Tabs)
// ==========================================

function MockTradeTabs() {
  const [activeTab, setActiveTab] = useState<string>("receive")

  const tabContent = {
    receive: {
      title: "积分收益 (Receive)",
      description: "本模块汇集用户所有的进账记录。可通过 API、活动赚取等形式增加积分值。",
      icon: Coins,
      color: "text-emerald-500",
      bgColor: "bg-emerald-500/10",
    },
    payment: {
      title: "积分消耗 (Payment)",
      description: "本模块汇总用户的出账流水。支持兑换、系统扣减与在线付款场景的费用记账。",
      icon: Gift,
      color: "text-orange-500",
      bgColor: "bg-orange-500/10",
    },
    community: {
      title: "社区划转 (Community)",
      description: "对接 Discourse 外部论坛同步时自动调度的积分余额转换事件。",
      icon: Shield,
      color: "text-purple-500",
      bgColor: "bg-purple-500/10",
    },
    online: {
      title: "在线流转 (Online)",
      description: "指代用户通过付款链接或集市网关主动进行的收单支付流转活动记录。",
      icon: Activity,
      color: "text-blue-500",
      bgColor: "bg-blue-500/10",
    },
    all: {
      title: "所有活动 (All Activities)",
      description: "汇总上述所有分类的完整历史日志，支持按分类检索和导出报表。",
      icon: Layers,
      color: "text-slate-500",
      bgColor: "bg-slate-500/10",
    },
  }

  const activeInfo = tabContent[activeTab as keyof typeof tabContent] || tabContent.receive

  return (
    <Card className="border border-dashed shadow-none rounded-lg p-5">
      <Tabs value={activeTab} onValueChange={setActiveTab} className="w-full">
        <TabsList className="flex p-0 gap-4 rounded-none w-full bg-transparent justify-start border-b border-border overflow-x-auto overflow-y-hidden">
          <TabsTrigger
            value="receive"
            className="data-[state=active]:bg-transparent data-[state=active]:shadow-none data-[state=active]:border-b-2 data-[state=active]:border-indigo-500 rounded-none border-b-2 border-transparent px-0 pb-2 text-xs font-bold text-muted-foreground data-[state=active]:text-indigo-500 -mb-[2px] transition-colors"
          >
            积分收益
          </TabsTrigger>
          <TabsTrigger
            value="payment"
            className="data-[state=active]:bg-transparent data-[state=active]:shadow-none data-[state=active]:border-b-2 data-[state=active]:border-indigo-500 rounded-none border-b-2 border-transparent px-0 pb-2 text-xs font-bold text-muted-foreground data-[state=active]:text-indigo-500 -mb-[2px] transition-colors"
          >
            积分消耗
          </TabsTrigger>
          <TabsTrigger
            value="community"
            className="data-[state=active]:bg-transparent data-[state=active]:shadow-none data-[state=active]:border-b-2 data-[state=active]:border-indigo-500 rounded-none border-b-2 border-transparent px-0 pb-2 text-xs font-bold text-muted-foreground data-[state=active]:text-indigo-500 -mb-[2px] transition-colors"
          >
            社区划转
          </TabsTrigger>
          <TabsTrigger
            value="online"
            className="data-[state=active]:bg-transparent data-[state=active]:shadow-none data-[state=active]:border-b-2 data-[state=active]:border-indigo-500 rounded-none border-b-2 border-transparent px-0 pb-2 text-xs font-bold text-muted-foreground data-[state=active]:text-indigo-500 -mb-[2px] transition-colors"
          >
            在线流转
          </TabsTrigger>
          <TabsTrigger
            value="all"
            className="data-[state=active]:bg-transparent data-[state=active]:shadow-none data-[state=active]:border-b-2 data-[state=active]:border-indigo-500 rounded-none border-b-2 border-transparent px-0 pb-2 text-xs font-bold text-muted-foreground data-[state=active]:text-indigo-500 -mb-[2px] transition-colors"
          >
            所有活动
          </TabsTrigger>
        </TabsList>

        <TabsContent value={activeTab} className="pt-4 animate-in fade-in duration-200">
          <div className="flex gap-4 items-start p-4 border border-dashed rounded-lg bg-muted/20">
            <div className={`size-10 rounded-lg flex items-center justify-center shrink-0 ${activeInfo.bgColor} ${activeInfo.color}`}>
              <activeInfo.icon className="size-5" />
            </div>
            <div className="space-y-1">
              <h4 className="text-sm font-semibold">{activeInfo.title}</h4>
              <p className="text-xs text-muted-foreground leading-normal">{activeInfo.description}</p>
            </div>
          </div>
        </TabsContent>
      </Tabs>
    </Card>
  )
}

// ==========================================
// 5. 代码片段复制辅助组件
// ==========================================

function CodeShowcase({ code, title }: { code: string; title: string }) {
  const [copied, setCopied] = useState(false)

  const handleCopy = async () => {
    try {
      await navigator.clipboard.writeText(code)
      setCopied(true)
      toast.success("代码片段已复制到剪贴板")
      setTimeout(() => setCopied(false), 2000)
    } catch {
      toast.error("复制失败")
    }
  }

  return (
    <Card className="border border-dashed shadow-none rounded-lg bg-slate-950 text-slate-200 overflow-hidden">
      <CardHeader className="py-2.5 px-4 bg-slate-900 border-b border-slate-800 flex flex-row items-center justify-between">
        <div className="text-[11px] font-mono text-slate-400 flex items-center gap-1.5">
          <Code2 className="size-3.5" />
          {title}
        </div>
        <Button variant="ghost" size="icon" className="size-6 text-slate-400 hover:text-white" onClick={handleCopy}>
          {copied ? <Check className="size-3 text-green-500" /> : <Copy className="size-3" />}
        </Button>
      </CardHeader>
      <pre className="p-4 font-mono text-[10px] overflow-x-auto leading-relaxed max-h-[300px] select-all [&::-webkit-scrollbar]:hidden">
        <code>{code}</code>
      </pre>
    </Card>
  )
}

// ==========================================
// 主组件 (Main Showcase)
// ==========================================

export default function ComponentLibraryPage() {
  const [activeCategory, setActiveCategory] = useState<string>("cards")

  const categories = [
    { id: "cards", title: "卡片组件 (Cards)", desc: "页面核心功能聚合引导卡片", icon: User },
    { id: "dashboards", title: "大盘统计 (Dashboards)", desc: "交易波动与客户数据展示图表", icon: Activity },
    { id: "tables", title: "数据表格 (Tables)", desc: "支持多类型与状态的明细流水表", icon: Layers },
    { id: "tabs", title: "多标签页 (Tabs)", desc: "分类管理多页签内容切换", icon: Info },
  ]

  const codeSnippets = useMemo(() => {
    return {
      cards: `// FeatureCard: 三栏网格功能展示卡片
interface FeatureCardProps {
  title: string;
  description: string;
  linkText: string;
  href?: string;
}

export function FeatureCard({ title, description, linkText, href }: FeatureCardProps) {
  return (
    <Card className="bg-card border border-dashed shadow-none hover:shadow-sm transition-all duration-300">
      <CardHeader className="pb-2">
        <CardTitle className="text-sm font-semibold">{title}</CardTitle>
      </CardHeader>
      <CardContent className="space-y-3">
        <CardDescription className="text-xs text-muted-foreground leading-normal">
          {description}
        </CardDescription>
        <Button variant="link" className="p-0 h-auto text-xs text-indigo-500" asChild>
          <a href={href || "#"}>{linkText} &rarr;</a>
        </Button>
      </CardContent>
    </Card>
  );
}`,
      dashboards: `// 7天收入统计: 进度条数据统计卡片
interface IncomeStatItem {
  date: string;
  amount: number;
  percentage: number;
}

export function IncomeStatsCard({ stats }: { stats: IncomeStatItem[] }) {
  return (
    <Card className="border border-dashed shadow-none p-5">
      <CardHeader className="p-0 pb-4">
        <CardTitle className="text-xs text-muted-foreground uppercase">7天收入统计</CardTitle>
      </CardHeader>
      <div className="space-y-2">
        {stats.map(s => (
          <div key={s.date} className="space-y-1">
            <div className="flex justify-between text-[10px]">
              <span className="text-muted-foreground">{s.date}</span>
              <span className="text-emerald-500 font-semibold">+{s.amount.toFixed(2)}</span>
            </div>
            <Progress value={s.percentage} className="h-1.5" />
          </div>
        ))}
      </div>
    </Card>
  );
}`,
      tables: `// 活动记录表格: 数据明细展现
interface MockOrder {
  id: string;
  order_name: string;
  amount: number;
  type: 'payment' | 'transfer' | 'community' | 'red_envelope';
  status: 'success' | 'pending' | 'expired' | 'disputing' | 'refund';
  order_no: string;
  created_at: string;
}

export function TransactionTable({ orders }: { orders: MockOrder[] }) {
  return (
    <table className="w-full text-xs">
      <thead>
        <tr className="border-b bg-muted/20">
          <th className="p-2 text-left font-medium">名称</th>
          <th className="p-2 text-center font-medium">额度</th>
          <th className="p-2 text-center font-medium">状态</th>
        </tr>
      </thead>
      <tbody>
        {orders.map(order => (
          <tr key={order.id} className="hover:bg-muted/30">
            <td className="p-2 font-medium">{order.order_name}</td>
            <td className="p-2 text-center font-semibold">{order.amount.toFixed(2)}</td>
            <td className="p-2 text-center">{order.status}</td>
          </tr>
        ))}
      </tbody>
    </table>
  );
}`,
      tabs: `// Tab 触发器及多模块切换导航
import { Tabs, TabsList, TabsTrigger, TabsContent } from "@/components/ui/tabs";

export function MockTradeTabs() {
  return (
    <Tabs defaultValue="receive" className="w-full">
      <TabsList className="flex gap-4 bg-transparent border-b">
        <TabsTrigger value="receive" className="px-0 pb-2 text-xs font-bold">
          积分收益
        </TabsTrigger>
        <TabsTrigger value="payment" className="px-0 pb-2 text-xs font-bold">
          积分消耗
        </TabsTrigger>
      </TabsList>
      <TabsContent value="receive">积分收益模块内容...</TabsContent>
      <TabsContent value="payment">积分消耗模块内容...</TabsContent>
    </Tabs>
  );
}`,
    }
  }, [])

  return (
    <div className="py-6 space-y-6 max-w-6xl mx-auto">
      <div>
        <h1 className="text-2xl font-bold tracking-tight">静态组件库 (Component Library)</h1>
        <p className="text-xs text-muted-foreground mt-1 leading-normal">
          展示项目重构中提取的有价值静态组件资源，提供可视化的静态演示与代码片段，便于平台后续的二次开发集成。
        </p>
      </div>

      <div className="grid grid-cols-1 lg:grid-cols-4 gap-6">
        {/* 左侧导航 */}
        <div className="lg:col-span-1 space-y-2">
          {categories.map((cat) => (
            <button
              key={cat.id}
              className={`w-full flex items-center gap-3 p-3 rounded-lg border text-left transition-all duration-200 cursor-pointer ${
                activeCategory === cat.id
                  ? "bg-indigo-500/10 border-indigo-500/30 text-indigo-600 dark:text-indigo-400 font-medium shadow-sm"
                  : "bg-card border-border hover:bg-muted/50 text-muted-foreground hover:text-foreground"
              }`}
              onClick={() => setActiveCategory(cat.id)}
            >
              <div className={`p-1.5 rounded-md ${activeCategory === cat.id ? "bg-indigo-500/20" : "bg-muted"}`}>
                <cat.icon className="size-4" />
              </div>
              <div className="min-w-0 flex-1">
                <div className="text-xs font-semibold">{cat.title}</div>
                <div className="text-[10px] text-muted-foreground truncate">{cat.desc}</div>
              </div>
            </button>
          ))}
        </div>

        {/* 右侧主展示区 */}
        <div className="lg:col-span-3 space-y-6">
          {activeCategory === "cards" && (
            <motion.div
              key="cards"
              initial={{ opacity: 0, y: 10 }}
              animate={{ opacity: 1, y: 0 }}
              transition={{ duration: 0.3 }}
              className="space-y-6"
            >
              <div>
                <h3 className="text-base font-semibold mb-3">卡片组件展示</h3>
                <ReceiveBanner />
                <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
                  <FeatureCard
                    title="接入官方 API 接口"
                    description="使用系统内置的 API 接口，支持快速接入、异步通知和同步跳转等标准流程。"
                    linkText="查看文档"
                    href="/docs/api"
                  />
                  <FeatureCard
                    title="探索开放平台"
                    description="通过开放平台页面浏览可用能力，快速对接通用系统功能。"
                    linkText="查看入口"
                    href="/docs/how-to-use"
                  />
                  <FeatureCard
                    title="面对面服务"
                    description="通过系统支持的离线与面对面服务接口，扩展应用功能，处理各种场景下的交互需求。"
                    linkText="功能开发中，敬请期待"
                  />
                </div>
              </div>

              <div className="space-y-2">
                <h3 className="text-xs font-semibold text-muted-foreground">组件代码片段 (FeatureCard)</h3>
                <CodeShowcase code={codeSnippets.cards} title="FeatureCard.tsx" />
              </div>
            </motion.div>
          )}

          {activeCategory === "dashboards" && (
            <motion.div
              key="dashboards"
              initial={{ opacity: 0, y: 10 }}
              animate={{ opacity: 1, y: 0 }}
              transition={{ duration: 0.3 }}
              className="space-y-6"
            >
              <div>
                <h3 className="text-base font-semibold mb-3">大盘数据组件展示</h3>
                <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                  <div className="space-y-4 md:col-span-2">
                    <MockTrendChart />
                  </div>
                  <MockIncomeStats />
                  <MockTopCustomers />
                </div>
              </div>

              <div className="space-y-2">
                <h3 className="text-xs font-semibold text-muted-foreground">组件代码片段 (IncomeStats)</h3>
                <CodeShowcase code={codeSnippets.dashboards} title="IncomeStatsCard.tsx" />
              </div>
            </motion.div>
          )}

          {activeCategory === "tables" && (
            <motion.div
              key="tables"
              initial={{ opacity: 0, y: 10 }}
              animate={{ opacity: 1, y: 0 }}
              transition={{ duration: 0.3 }}
              className="space-y-6"
            >
              <div>
                <h3 className="text-base font-semibold mb-3">活动明细表格展示</h3>
                <MockTransactionTable />
              </div>

              <div className="space-y-2">
                <h3 className="text-xs font-semibold text-muted-foreground">组件代码片段 (TransactionTable)</h3>
                <CodeShowcase code={codeSnippets.tables} title="TransactionTable.tsx" />
              </div>
            </motion.div>
          )}

          {activeCategory === "tabs" && (
            <motion.div
              key="tabs"
              initial={{ opacity: 0, y: 10 }}
              animate={{ opacity: 1, y: 0 }}
              transition={{ duration: 0.3 }}
              className="space-y-6"
            >
              <div>
                <h3 className="text-base font-semibold mb-3">多标签导航展示</h3>
                <MockTradeTabs />
              </div>

              <div className="space-y-2">
                <h3 className="text-xs font-semibold text-muted-foreground">组件代码片段 (MockTradeTabs)</h3>
                <CodeShowcase code={codeSnippets.tabs} title="TradeTabs.tsx" />
              </div>
            </motion.div>
          )}
        </div>
      </div>
    </div>
  )
}
