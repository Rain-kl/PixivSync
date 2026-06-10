import {useQuery} from "@tanstack/react-query"

import {PixezService} from "@/lib/services"

export function usePixEzAccounts() {
  return useQuery({
    queryKey: ["pixez", "accounts"],
    queryFn: () => PixezService.listAccounts(),
  })
}
