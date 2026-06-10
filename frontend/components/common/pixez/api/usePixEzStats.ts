import {useQuery} from "@tanstack/react-query"

import {PixezService} from "@/lib/services"

export function usePixEzStats() {
  return useQuery({
    queryKey: ["pixez", "dashboard"],
    queryFn: () => PixezService.getDashboard(),
  })
}
