"use client"

import type {FormEvent} from "react"
import {useEffect, useState} from "react"
import {useQueryClient} from "@tanstack/react-query"
import {Gauge, Save} from "lucide-react"
import {toast} from "sonner"

import {Button} from "@/components/ui/button"
import {Card, CardContent, CardDescription, CardHeader, CardTitle} from "@/components/ui/card"
import {Field, FieldGroup, FieldLabel} from "@/components/ui/field"
import {Input} from "@/components/ui/input"
import {Skeleton} from "@/components/ui/skeleton"
import {Spinner} from "@/components/ui/spinner"
import {ErrorInline} from "@/components/layout/error"
import {AdminService} from "@/lib/services"

import {PIXEZ_RATE_LIMIT_CONFIG_KEYS, usePixEzSettings} from "./api/usePixEzSettings"

type FormState = {
  downloadInterval: string
  illustConcurrency: string
  novelConcurrency: string
}

const fallbackValues: FormState = {
  downloadInterval: "1",
  illustConcurrency: "5",
  novelConcurrency: "5",
}

function configValue(configs: Record<string, string>, key: string, fallback: string) {
  return configs[key] ?? fallback
}

function validateForm(values: FormState) {
  const interval = Number(values.downloadInterval)
  const illust = Number(values.illustConcurrency)
  const novel = Number(values.novelConcurrency)

  if (!Number.isFinite(interval) || interval < 0) return "多图下载间隔必须大于等于 0"
  if (!Number.isInteger(illust) || illust < 1 || illust > 50) return "插画并发上限必须是 1 到 50 的整数"
  if (!Number.isInteger(novel) || novel < 1 || novel > 50) return "小说并发上限必须是 1 到 50 的整数"

  return null
}

export function PixEzRateLimitForm() {
  const queryClient = useQueryClient()
  const settingsQuery = usePixEzSettings()
  const [saving, setSaving] = useState(false)
  const [values, setValues] = useState<FormState>(fallbackValues)

  useEffect(() => {
    setValues({
      downloadInterval: configValue(
        settingsQuery.configs,
        PIXEZ_RATE_LIMIT_CONFIG_KEYS.downloadInterval,
        fallbackValues.downloadInterval,
      ),
      illustConcurrency: configValue(
        settingsQuery.configs,
        PIXEZ_RATE_LIMIT_CONFIG_KEYS.illustConcurrency,
        fallbackValues.illustConcurrency,
      ),
      novelConcurrency: configValue(
        settingsQuery.configs,
        PIXEZ_RATE_LIMIT_CONFIG_KEYS.novelConcurrency,
        fallbackValues.novelConcurrency,
      ),
    })
  }, [settingsQuery.configs])

  const updateValue = (key: keyof FormState, value: string) => {
    setValues((current) => ({...current, [key]: value}))
  }

  const handleSubmit = async (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault()
    const validationError = validateForm(values)
    if (validationError) {
      toast.error(validationError)
      return
    }

    try {
      setSaving(true)
      await Promise.all([
        AdminService.updateSystemConfig(PIXEZ_RATE_LIMIT_CONFIG_KEYS.downloadInterval, {
          value: values.downloadInterval,
        }),
        AdminService.updateSystemConfig(PIXEZ_RATE_LIMIT_CONFIG_KEYS.illustConcurrency, {
          value: values.illustConcurrency,
        }),
        AdminService.updateSystemConfig(PIXEZ_RATE_LIMIT_CONFIG_KEYS.novelConcurrency, {
          value: values.novelConcurrency,
        }),
      ])
      await Promise.all([
        queryClient.invalidateQueries({queryKey: ["admin", "system-configs"]}),
        queryClient.invalidateQueries({queryKey: ["public-config"]}),
        queryClient.invalidateQueries({queryKey: ["pixez", "dashboard"]}),
      ])
      toast.success("抓取限制配置已热更新，下一次下载任务起生效")
    } catch (error) {
      toast.error("保存抓取限制失败", {
        description: error instanceof Error ? error.message : "未知错误",
      })
    } finally {
      setSaving(false)
    }
  }

  return (
    <Card className="border border-dashed shadow-sm">
      <CardHeader className="border-b border-dashed pb-4">
        <div className="flex items-center justify-between gap-4">
          <div className="flex items-center gap-2">
            <div className="p-1.5 rounded-lg bg-indigo-500/10 text-indigo-500">
              <Gauge className="size-4" />
            </div>
            <div>
              <CardTitle className="text-base font-semibold">PixEz 抓取限制</CardTitle>
              <CardDescription className="text-xs">配置 Pixiv 收藏同步的并发下载上限与时间间隔限制</CardDescription>
            </div>
          </div>
          {!settingsQuery.isLoading && !settingsQuery.error && (
            <Button
              type="submit"
              form="pixez-rate-limit-form"
              disabled={saving}
              size="sm"
              variant={'secondary'}
              className="gap-1.5"
            >
              {saving ? <Spinner className="size-3.5" /> : <Save className="size-3.5" />}
              <span>保存</span>
            </Button>
          )}
        </div>
      </CardHeader>
      <CardContent className="pt-6">
        {settingsQuery.isLoading ? (
          <div className="flex flex-col gap-5">
            <Skeleton className="h-16 w-full" />
            <Skeleton className="h-16 w-full" />
            <Skeleton className="h-16 w-full" />
          </div>
        ) : settingsQuery.error ? (
          <ErrorInline error={settingsQuery.error} onRetry={() => settingsQuery.refetch()} />
        ) : (
          <form id="pixez-rate-limit-form" onSubmit={handleSubmit}>
            <FieldGroup>
              <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                <Field>
                  <FieldLabel htmlFor="pixez-download-interval">多图下载间隔时间</FieldLabel>
                  <Input
                    id="pixez-download-interval"
                    type="number"
                    min="0"
                    step="1"
                    placeholder="单位秒，默认 1"
                    value={values.downloadInterval}
                    onChange={(event) => updateValue("downloadInterval", event.target.value)}
                  />
                </Field>

                <Field>
                  <FieldLabel htmlFor="pixez-illust-concurrency">插画并发下载上限</FieldLabel>
                  <Input
                    id="pixez-illust-concurrency"
                    type="number"
                    min="1"
                    max="50"
                    step="1"
                    placeholder="建议 3 到 10，默认 5"
                    value={values.illustConcurrency}
                    onChange={(event) => updateValue("illustConcurrency", event.target.value)}
                  />
                </Field>

                <Field>
                  <FieldLabel htmlFor="pixez-novel-concurrency">小说并发下载上限</FieldLabel>
                  <Input
                    id="pixez-novel-concurrency"
                    type="number"
                    min="1"
                    max="50"
                    step="1"
                    placeholder="建议 3 到 10，默认 5"
                    value={values.novelConcurrency}
                    onChange={(event) => updateValue("novelConcurrency", event.target.value)}
                  />
                </Field>
              </div>
            </FieldGroup>
          </form>
        )}
      </CardContent>
    </Card>
  )
}
