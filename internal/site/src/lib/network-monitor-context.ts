import { createContext, useContext } from "react"
import type { ChartTimes, NetworkMonitorRecord, RawMonitorStatsRecord } from "@/types"
export type NetworkMonitorAdapter = {
	canManage: boolean
	list: () => Promise<NetworkMonitorRecord[]>
	stats: (ids: string[], range: ChartTimes) => Promise<RawMonitorStatsRecord[]>
	create: (data: Partial<NetworkMonitorRecord>) => Promise<unknown>
	update: (id: string, data: Partial<NetworkMonitorRecord>) => Promise<unknown>
	remove: (id: string) => Promise<unknown>
}
export const NetworkMonitorContext = createContext<NetworkMonitorAdapter | null>(null)
export const useNetworkMonitorAdapter = () => useContext(NetworkMonitorContext)
