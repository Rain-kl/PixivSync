import type {PixezMirrorStatusText, PixezMirrorTarget, PixezRunStatus} from "@/lib/services"

const numberFormatter = new Intl.NumberFormat("zh-CN")

export function formatPixEzNumber(value: number | undefined) {
  return numberFormatter.format(value ?? 0)
}

export function formatPixEzPercent(value: number | undefined) {
  return `${(value ?? 0).toFixed(1)}%`
}

export function formatPixEzDateTime(value?: string | null) {
  if (!value) return "-"
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return value

  return new Intl.DateTimeFormat("zh-CN", {
    year: "numeric",
    month: "2-digit",
    day: "2-digit",
    hour: "2-digit",
    minute: "2-digit",
    hour12: false,
  }).format(date)
}

export function formatPixEzDuration(milliseconds: number | undefined) {
  const value = milliseconds ?? 0
  if (value <= 0) return "-"
  if (value < 1000) return `${value}ms`

  const seconds = Math.round(value / 1000)
  if (seconds < 60) return `${seconds}s`

  const minutes = Math.floor(seconds / 60)
  const restSeconds = seconds % 60
  return restSeconds > 0 ? `${minutes}m${restSeconds}s` : `${minutes}m`
}

export function formatPixEzFileSize(bytes: number | undefined) {
  const value = bytes ?? 0
  if (value <= 0) return "-"
  if (value < 1024) return `${value} B`
  if (value < 1024 * 1024) return `${(value / 1024).toFixed(1)} KB`
  return `${(value / 1024 / 1024).toFixed(1)} MB`
}

export function pixezTargetLabel(target: PixezMirrorTarget) {
  return target === "illust" ? "插画" : "小说"
}

export function pixezRunStatusLabel(status: PixezRunStatus | string) {
  if (status === "running") return "运行中"
  if (status === "success") return "成功"
  if (status === "failed") return "失败"
  return status || "-"
}

export function pixezMirrorStatusLabel(status: PixezMirrorStatusText | string) {
  if (status === "success") return "成功"
  if (status === "processing") return "下载中"
  if (status === "failed") return "失败"
  if (status === "none") return "未入队"
  return status || "-"
}

export type PixezImageQuality = "low" | "medium" | "high" | "origin"

export function mirrorImageURL(url: string | undefined, quality: PixezImageQuality = "origin") {
  if (!url) return ""
  const mirrorURL = url
    .replace("https://i.pximg.net", "/mirror/pximg")
    .replace("https://s.pximg.net", "/mirror/pximg")
  if (quality === "origin") return mirrorURL

  const separator = mirrorURL.includes("?") ? "&" : "?"
  return `${mirrorURL}${separator}quality=${quality}`
}
