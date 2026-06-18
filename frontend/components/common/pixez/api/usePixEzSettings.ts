import {useMemo} from "react"
import {useQuery} from "@tanstack/react-query"

import {AdminSystemConfigService} from "@/lib/services"

export const PIXEZ_RATE_LIMIT_CONFIG_KEYS = {
  downloadInterval: "pixez_mirror_download_interval_seconds",
  illustConcurrency: "pixez_mirror_illust_concurrency",
  novelConcurrency: "pixez_mirror_novel_concurrency",
} as const

export function usePixEzSettings() {
  const query = useQuery({
    queryKey: ["admin", "system-configs", "business"],
    queryFn: () => AdminSystemConfigService.listSystemConfigs("business"),
  })

  const configs = useMemo(() => {
    return (query.data ?? []).reduce<Record<string, string>>((accumulator, config) => {
      accumulator[config.key] = config.value
      return accumulator
    }, {})
  }, [query.data])

  return {
    ...query,
    configs,
  }
}
