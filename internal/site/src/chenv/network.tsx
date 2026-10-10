import { useEffect, useMemo, useState } from "react"
import NetworkMonitorsTable from "@/components/network-monitors-table/network-monitors-table"
import { NetworkMonitorContext, type NetworkMonitorAdapter } from "@/lib/network-monitor-context"
import { getPbTimestamp } from "@/lib/api"
import { chartTimeData } from "@/lib/utils"
import type { NetworkMonitorRecord, RawMonitorStatsRecord } from "@/types"
import { admin, adminRequest, publicGet } from "./api"

export default function ChenvNetwork({ systemId, adminMode = false }: { systemId?: string; adminMode?: boolean }) {
	const [monitors, setMonitors] = useState<NetworkMonitorRecord[]>([])
	const [loading, setLoading] = useState(true)
	const [error, setError] = useState("")
	const [revision, setRevision] = useState(0)
	const adapter = useMemo<NetworkMonitorAdapter>(() => {
		const list = async () => {
			const records = await (adminMode
				? adminRequest<NetworkMonitorRecord[]>("monitors")
				: publicGet<NetworkMonitorRecord[]>("monitors"))
			return records.map((m) => ({
				...m,
				target: m.target || "",
				server: m.server || "",
				port: m.port || 0,
				interval: m.interval || 30,
				name: m.name,
			}))
		}
		const save = async (id: string | undefined, data: Partial<NetworkMonitorRecord>) => {
			let current: Partial<NetworkMonitorRecord> = {
				enabled: true,
				public: false,
				name: "",
				interval: 30,
				port: 0,
				server: "",
			}
			if (id) {
				const rows = await adminRequest<NetworkMonitorRecord[]>("monitors")
				current = rows.find((m) => m.id === id) || {}
				if (!current.id) throw new Error("探测已被删除")
			}
			const result = await adminRequest("monitors" + (id ? "/" + id : ""), id ? "PUT" : "POST", { ...current, ...data })
			setRevision((v) => v + 1)
			return result
		}
		return {
			canManage: adminMode,
			list,
			stats: async (ids, range) =>
				adminMode
					? admin
							.collection<RawMonitorStatsRecord>("network_monitor_stats")
							.getFullList({
								filter: admin.filter(
									"(" + ids.map((_, i) => `monitor={:m${i}}`).join(" || ") + ") && type={:type} && created>{:created}",
									{
										...Object.fromEntries(ids.map((id, i) => ["m" + i, id])),
										type: chartTimeData[range].type,
										created: getPbTimestamp(range, undefined, true),
									}
								),
								sort: "created",
								fields: "monitor,created,res_min,res_max,res_sum,total_count,success_count",
							})
					: publicGet<RawMonitorStatsRecord[]>(`monitor-stats?ids=${ids.join(",")}&range=${range}`),
			create: (data) => save(undefined, data),
			update: (id, data) => save(id, data),
			remove: async (id) => {
				const result = await adminRequest("monitors/" + id, "DELETE")
				setRevision((v) => v + 1)
				return result
			},
		}
	}, [adminMode])
	useEffect(() => {
		let active = true
		let timer: ReturnType<typeof setTimeout>
		const load = async () => {
			try {
				const rows = await adapter.list()
				if (active) {
					setMonitors(systemId ? rows.filter((m) => m.system === systemId) : rows)
					setError("")
				}
			} catch (e) {
				if (active) setError((e as Error).message)
			} finally {
				if (active) {
					setLoading(false)
					timer = setTimeout(load, 10000)
				}
			}
		}
		void load()
		return () => {
			active = false
			clearTimeout(timer)
		}
	}, [adapter, revision, systemId])
	return (
		<NetworkMonitorContext.Provider value={adapter}>
			{error && (
				<p role="alert" className="text-sm text-destructive mb-4">
					{error}
				</p>
			)}
			<NetworkMonitorsTable systemId={systemId} monitors={monitors} isLoading={loading} />
		</NetworkMonitorContext.Provider>
	)
}
