import { useMemo, useState } from "react"
import { useStore } from "@nanostores/react"
import InfoBar from "@/components/routes/system/info-bar"
import { CpuChart } from "@/components/routes/system/charts/cpu-charts"
import { MemoryChart, SwapChart } from "@/components/routes/system/charts/memory-charts"
import { RootDiskCharts, DiskUsageChart, DiskIOChart } from "@/components/routes/system/charts/disk-charts"
import { BandwidthChart } from "@/components/routes/system/charts/network-charts"
import { LoadAverageChart } from "@/components/routes/system/charts/load-average-chart"
import type { SystemData } from "@/components/routes/system/use-system-data"
import ChenvNetwork from "./network"
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs"
import { $chartTime, $maxValues } from "@/lib/stores"
import { parseSemVer } from "@/lib/utils"
import type { ChartData, SystemStatsRecord, SystemRecord, SystemDetailsRecord } from "@/types"
import type { System } from "./api"

// Adapt the anonymous, filtered response to Beszel's existing chart components.
// The original authenticated data hook is intentionally not mounted here.
export default function PublicDetail({ system }: { system: System }) {
	const chartTime = useStore($chartTime)
	const maxValues = useStore($maxValues)
	const [grid, setGridState] = useState(() => localStorage.getItem("besz-grid") !== "false")
	const [displayMode, setDisplayModeState] = useState<"default" | "tabs">(() =>
		localStorage.getItem("besz-displayMode") === '"tabs"' ? "tabs" : "default"
	)
	const setGrid = (v: boolean) => {
		setGridState(v)
		localStorage.setItem("besz-grid", JSON.stringify(v))
	}
	const setDisplayMode = (v: "default" | "tabs") => {
		setDisplayModeState(v)
		localStorage.setItem("besz-displayMode", JSON.stringify(v))
	}
	const systemStats = useMemo(
		() =>
			(system.history || []).map((row) => ({
				...row,
				created: new Date(row.created).getTime(),
			})) as unknown as SystemStatsRecord[],
		[system.history]
	)
	const record = { ...system, host: system.host || "", port: system.port || "" } as unknown as SystemRecord
	const chartData: ChartData = {
		systemStats,
		containerData: [],
		chartTime,
		orientation: "left",
		agentVersion: parseSemVer(record.info.v || "0.21.0"),
	}
	const dataEmpty = systemStats.length < 2
	const isLongerChart = !["1h", "1m"].includes(chartTime)
	const coreProps = { chartData, grid, dataEmpty, maxValues, isLongerChart, showMax: maxValues && isLongerChart }
	const diskData = { ...coreProps, system: record } as SystemData
	const extraFilesystems = [...new Set(systemStats.flatMap((row) => Object.keys(row.stats.efs || {})))]
	const extraDisks = extraFilesystems.map((name) => (
		<div key={name} className="contents">
			<DiskUsageChart systemData={diskData} extraFsName={name} />
			<DiskIOChart systemData={diskData} extraFsName={name} />
		</div>
	))
	const cpuMemory = (
		<>
			<CpuChart {...coreProps} />
			<MemoryChart {...coreProps} />
			<SwapChart {...coreProps} systemStats={systemStats} />
			<LoadAverageChart {...coreProps} />
		</>
	)
	return (
		<div className="grid gap-4">
			<InfoBar
				system={record}
				chartData={chartData}
				grid={grid}
				setGrid={setGrid}
				displayMode={displayMode}
				setDisplayMode={setDisplayMode}
				details={(system.hardware || null) as SystemDetailsRecord | null}
				allowRealtime={false}
			/>
			{displayMode === "default" ? (
				<>
					<div className="grid xl:grid-cols-2 gap-4">
						<CpuChart {...coreProps} />
						<MemoryChart {...coreProps} />
						<RootDiskCharts systemData={diskData} />
						{extraDisks}
						<BandwidthChart {...coreProps} systemStats={systemStats} />
						<SwapChart {...coreProps} systemStats={systemStats} />
						<LoadAverageChart {...coreProps} />
					</div>
					<ChenvNetwork systemId={system.id} />
				</>
			) : (
				<Tabs defaultValue="core" className="contents">
					<TabsList className="h-11 p-1.5 w-full shadow-xs">
						<TabsTrigger value="core" className="w-full">
							核心
						</TabsTrigger>
						<TabsTrigger value="network" className="w-full">
							网络
						</TabsTrigger>
						<TabsTrigger value="disk" className="w-full">
							磁盘
						</TabsTrigger>
					</TabsList>
					<TabsContent value="core">
						<div className="grid xl:grid-cols-2 gap-4">{cpuMemory}</div>
					</TabsContent>
					<TabsContent value="network">
						<div className="grid gap-4">
							<BandwidthChart {...coreProps} systemStats={systemStats} />
							<ChenvNetwork systemId={system.id} />
						</div>
					</TabsContent>
					<TabsContent value="disk">
						<div className="grid xl:grid-cols-2 gap-4">
							<RootDiskCharts systemData={diskData} />
							{extraDisks}
						</div>
					</TabsContent>
				</Tabs>
			)}
		</div>
	)
}
