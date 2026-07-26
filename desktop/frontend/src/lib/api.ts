import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'

import {
  GetAccounts,
  GetBreakdown,
  GetConfig,
  GetCurrentAccount,
  GetHealth,
  GetModelsLatest,
  GetSnapshots,
  GetSparklines,
  GetStats,
  GetStatus,
  GetTimeline,
  GetTimeseries,
  SetConfig,
} from '../../wailsjs/go/main/App'
import type { config } from '../../wailsjs/go/models'

// The monitor polls language servers every 60s; refreshing at half that keeps
// the UI at most one poll cycle behind without hammering the loopback API.
const REFETCH_MS = 30_000
// Live monitor state (ACTIVE/IDLE) is cheap and changes fast; poll it quicker.
const STATUS_REFETCH_MS = 10_000

/** True when the app cannot reach its local monitoring runtime. */
export function isDaemonUnreachable(error: unknown): boolean {
  return String(error).includes('daemon unreachable')
}

export function useHealth() {
  return useQuery({
    queryKey: ['health'],
    queryFn: GetHealth,
    refetchInterval: STATUS_REFETCH_MS,
    retry: false,
  })
}

export function useDaemonStatus() {
  return useQuery({
    queryKey: ['status'],
    queryFn: GetStatus,
    refetchInterval: STATUS_REFETCH_MS,
    retry: false,
  })
}

export function useCurrentAccount() {
  return useQuery({
    queryKey: ['currentAccount'],
    queryFn: GetCurrentAccount,
    refetchInterval: STATUS_REFETCH_MS,
    retry: false,
  })
}

export function useAccounts() {
  return useQuery({
    queryKey: ['accounts'],
    queryFn: GetAccounts,
    refetchInterval: REFETCH_MS,
    retry: false,
  })
}

export function useModelsLatest() {
  return useQuery({
    queryKey: ['modelsLatest'],
    queryFn: GetModelsLatest,
    refetchInterval: REFETCH_MS,
    retry: false,
  })
}

export function useSnapshots(email: string, limit = 10) {
  return useQuery({
    queryKey: ['snapshots', email, limit],
    queryFn: () => GetSnapshots(email, limit, ''),
    enabled: email !== '',
    refetchInterval: REFETCH_MS,
    retry: false,
  })
}

export function useSparklines(email: string) {
  return useQuery({
    queryKey: ['sparklines', email],
    queryFn: () => GetSparklines(email),
    enabled: email !== '',
    refetchInterval: REFETCH_MS,
    retry: false,
  })
}

export function useTimeline(email: string) {
  return useQuery({
    queryKey: ['timeline', email],
    queryFn: () => GetTimeline(email),
    enabled: email !== '',
    refetchInterval: REFETCH_MS,
    retry: false,
  })
}

export function useTimeseries(range: '7d' | '30d', agg: 'avg' | 'min') {
  return useQuery({
    queryKey: ['timeseries', range, agg],
    queryFn: () => GetTimeseries(range, agg),
    refetchInterval: REFETCH_MS,
    retry: false,
  })
}

export function useBreakdown() {
  return useQuery({
    queryKey: ['breakdown'],
    queryFn: GetBreakdown,
    refetchInterval: REFETCH_MS,
    retry: false,
  })
}

export function useStats() {
  return useQuery({
    queryKey: ['stats'],
    queryFn: GetStats,
    refetchInterval: REFETCH_MS,
    retry: false,
  })
}

export function useAppConfig() {
  return useQuery({
    queryKey: ['config'],
    queryFn: GetConfig,
    staleTime: Infinity,
  })
}

export function useSetAppConfig() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (cfg: config.Config) => SetConfig(cfg),
    onSuccess: (saved) => {
      queryClient.setQueryData(['config'], saved)
      // The local API settings may have changed; refresh all monitor data.
      queryClient.invalidateQueries()
    },
  })
}
