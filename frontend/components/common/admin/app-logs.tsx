/*
Copyright 2026 Arctel.net

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

"use client"

import {useCallback, useEffect, useRef, useState} from "react"
import {toast} from "sonner"
import {ArrowDown, ChevronUp, Loader2, Pause, Play} from "lucide-react"

import {AdminService} from "@/lib/services"
import {ErrorInline} from "@/components/layout/error"
import {LoadingStateWithBorder} from "@/components/layout/loading"
import {Badge} from "@/components/ui/badge"
import {Button} from "@/components/ui/button"

interface LogEntry {
  index: number
  data: string
}

// Distance (px) from bottom to treat as "at bottom"
const BOTTOM_THRESHOLD = 40

function getApiBaseUrl(): string {
  if (typeof window !== "undefined") {
    // If NEXT_PUBLIC_WAVELET_BACKEND_URL is set, use it. Otherwise, use origin.
    const base = process.env.NEXT_PUBLIC_WAVELET_BACKEND_URL || ""
    if (base.startsWith("http")) return base
    
    // Relative URL fallback
    const proto = window.location.protocol
    const host = window.location.host
    return `${proto}//${host}${base}`
  }
  return process.env.NEXT_PUBLIC_WAVELET_BACKEND_URL || ""
}

function buildWsUrl(): string {
  const base = getApiBaseUrl()
  const wsBase = base.replace(/^http/, "ws")
  return `${wsBase}/api/v1/admin/logs/ws`
}

function parseLogLevel(line: string): "debug" | "info" | "warn" | "error" | "unknown" {
  const lower = line.toLowerCase()
  if (lower.includes("\"level\":\"error\"") || lower.includes("level=error")) return "error"
  if (lower.includes("\"level\":\"warn\"") || lower.includes("level=warn")) return "warn"
  if (lower.includes("\"level\":\"debug\"") || lower.includes("level=debug")) return "debug"
  if (lower.includes("\"level\":\"info\"") || lower.includes("level=info")) return "info"
  return "unknown"
}

export function AppLogs() {
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<Error | null>(null)

  const [logs, setLogs] = useState<LogEntry[]>([])
  const [hasMore, setHasMore] = useState(false)
  const [nextCursor, setNextCursor] = useState(0)
  const [loadingMore, setLoadingMore] = useState(false)

  const [connected, setConnected] = useState(false)
  const [paused, setPaused] = useState(false)

  // autoScroll = true → new logs auto-scroll to bottom
  const [autoScroll, setAutoScroll] = useState(true)

  const containerRef = useRef<HTMLDivElement>(null)
  const wsRef = useRef<WebSocket | null>(null)
  const pausedRef = useRef(paused)
  const autoScrollRef = useRef(autoScroll)
  const isUserScrolling = useRef(false)

  useEffect(() => { pausedRef.current = paused }, [paused])
  useEffect(() => { autoScrollRef.current = autoScroll }, [autoScroll])

  // ---- Scroll detection ------------------------------------------------
  const handleScroll = useCallback(() => {
    const el = containerRef.current
    if (!el) return

    if (!isUserScrolling.current) return

    const atBottom = el.scrollHeight - el.scrollTop - el.clientHeight < BOTTOM_THRESHOLD
    if (atBottom && !autoScrollRef.current) {
      setAutoScroll(true)
    } else if (!atBottom && autoScrollRef.current) {
      setAutoScroll(false)
    }
  }, [])

  const handleWheel = useCallback(() => { isUserScrolling.current = true }, [])
  const handleTouchStart = useCallback(() => { isUserScrolling.current = true }, [])

  useEffect(() => {
    const el = containerRef.current
    if (!el) return

    let timer: ReturnType<typeof setTimeout>
    const onScrollEnd = () => {
      clearTimeout(timer)
      timer = setTimeout(() => { isUserScrolling.current = false }, 150)
    }
    el.addEventListener("scroll", onScrollEnd, { passive: true })
    return () => {
      el.removeEventListener("scroll", onScrollEnd)
      clearTimeout(timer)
    }
  }, [])

  // ---- Auto-scroll to bottom when new logs arrive ----------------------
  useEffect(() => {
    if (!autoScroll || !containerRef.current) return
    isUserScrolling.current = false
    const el = containerRef.current
    requestAnimationFrame(() => {
      el.scrollTop = el.scrollHeight
    })
  }, [logs, autoScroll])

  // ---- Data fetching ---------------------------------------------------
  const fetchLogs = useCallback(async (cursor: number = 0) => {
    try {
      return await AdminService.getLogs(cursor)
    } catch (err) {
      throw err instanceof Error ? err : new Error("获取日志失败")
    }
  }, [])

  const loadHistory = useCallback(async (cursor: number = 0) => {
    const isInitial = cursor === 0
    if (isInitial) {
      setLoading(true)
      setError(null)
    } else {
      setLoadingMore(true)
    }

    try {
      const data = await fetchLogs(cursor)
      if (isInitial) {
        setLogs(data.lines || [])
      } else {
        setLogs(prev => [...(data.lines || []), ...prev])

        requestAnimationFrame(() => {
          const el = containerRef.current
          if (!el) return
          const newCount = (data.lines || []).length
          const lineH = 20
          isUserScrolling.current = false
          el.scrollTop = el.scrollTop + newCount * lineH
        })
      }
      setHasMore(data.has_more)
      setNextCursor(data.next_cursor)
    } catch (err) {
      if (isInitial) {
        setError(err instanceof Error ? err : new Error("获取日志失败"))
      } else {
        toast.error("加载更早日志失败")
      }
    } finally {
      if (isInitial) setLoading(false)
      else setLoadingMore(false)
    }
  }, [fetchLogs])

  // ---- WebSocket -------------------------------------------------------
  const connectWs = useCallback(() => {
    if (wsRef.current) wsRef.current.close()

    const ws = new WebSocket(buildWsUrl())
    wsRef.current = ws

    ws.onopen = () => { setConnected(true) }

    ws.onmessage = (event) => {
      if (pausedRef.current) return
      try {
        const msg = JSON.parse(event.data)
        if (msg.type === "log" && msg.data) {
          const entry: LogEntry = msg.data
          setLogs(prev => {
            const next = [...prev, entry]
            return next.length > 2000 ? next.slice(-2000) : next
          })
        }
      } catch { /* ignore */ }
    }

    ws.onclose = () => { setConnected(false); wsRef.current = null }
    ws.onerror = () => { setConnected(false) }
  }, [])

  // ---- Initialize ------------------------------------------------------
  useEffect(() => {
    loadHistory(0).then(() => connectWs())
    return () => {
      wsRef.current?.close()
      wsRef.current = null
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [])

  // ---- Actions ---------------------------------------------------------
  const scrollToBottom = useCallback(() => {
    setAutoScroll(true)
    requestAnimationFrame(() => {
      if (containerRef.current) {
        containerRef.current.scrollTop = containerRef.current.scrollHeight
      }
    })
  }, [])

  const togglePause = useCallback(() => setPaused(p => !p), [])
  const reconnect = useCallback(() => connectWs(), [connectWs])
  const handleLoadMore = useCallback(() => {
    if (nextCursor > 0) loadHistory(nextCursor)
  }, [nextCursor, loadHistory])

  // ---- Render ----------------------------------------------------------
  if (loading) return <LoadingStateWithBorder />
  if (error) return <ErrorInline error={error} onRetry={() => loadHistory(0)} />

  return (
    <div className="flex flex-col h-full relative">
      {/* Sub Header / Control Bar */}
      <div className="flex items-center justify-between pb-3 border-b border-border/40 mb-3">
        <div className="text-sm text-muted-foreground font-medium">
          系统后台实时输出的运行日志 (最多缓存 2000 行)
        </div>
        <div className="flex items-center gap-2">
          <Badge variant={connected ? "secondary" : "destructive"} className="h-6">
            {connected ? "已连接" : "断开连接"}
          </Badge>
          {connected && (
            <Button variant="outline" size="sm" onClick={togglePause} className="h-8">
              {paused
                ? <><Play className="size-3.5 mr-1.5" />恢复</>
                : <><Pause className="size-3.5 mr-1.5" />暂停</>
              }
            </Button>
          )}
          {!connected && (
            <Button variant="outline" size="sm" onClick={reconnect} className="h-8">
              重新连接
            </Button>
          )}
        </div>
      </div>

      {/* Log viewer — fixed height, scrollable */}
      <div
        ref={containerRef}
        onScroll={handleScroll}
        onWheel={handleWheel}
        onTouchStart={handleTouchStart}
        className="h-[calc(100vh-270px)] overflow-y-auto overflow-x-hidden rounded-md border border-border/50 bg-[#0d1117] font-mono text-[13px] leading-5 relative"
      >
        {/* Load older logs */}
        {hasMore && (
          <div className="sticky top-0 z-10 flex justify-center py-1.5 bg-[#0d1117]/90 backdrop-blur-sm">
            <Button
              variant="ghost"
              size="sm"
              onClick={handleLoadMore}
              disabled={loadingMore}
              className="text-muted-foreground hover:text-foreground h-7 text-xs"
            >
              {loadingMore
                ? <><Loader2 className="size-3 mr-1.5 animate-spin" />加载中...</>
                : <><ChevronUp className="size-3 mr-1.5" />加载更早日志</>
              }
            </Button>
          </div>
        )}

        {/* Log lines */}
        <div className="px-3 py-2">
          {logs.length === 0 ? (
            <div className="text-center text-gray-500 py-8">暂无日志</div>
          ) : (
            logs.map((entry) => {
              const level = parseLogLevel(entry.data)
              const color = level === "error"
                ? "text-red-400"
                : level === "warn"
                  ? "text-yellow-400"
                  : level === "debug"
                    ? "text-gray-500"
                    : "text-gray-300"
              return (
                <div
                  key={entry.index}
                  className={`${color} whitespace-pre-wrap break-all hover:bg-white/5`}
                >
                  {entry.data}
                </div>
              )
            })
          )}
        </div>
      </div>

      {/* Floating "back to latest" button */}
      {!autoScroll && (
        <div className="absolute bottom-6 left-1/2 -translate-x-1/2 z-20">
          <Button
            variant="outline"
            size="sm"
            onClick={scrollToBottom}
            className="shadow-lg bg-background/80 backdrop-blur-sm border-border/50"
          >
            <ArrowDown className="size-3.5 mr-1.5" />
            回到最新
          </Button>
        </div>
      )}
    </div>
  )
}
