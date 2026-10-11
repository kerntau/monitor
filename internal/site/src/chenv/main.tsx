import React, { lazy, Suspense, useEffect, useState } from "react"
import { createRoot } from "react-dom/client"
import { i18n } from "@lingui/core"
import { I18nProvider } from "@lingui/react"
import { useStore } from "@nanostores/react"
import { GithubIcon, LoaderCircle, Server, RefreshCw } from "lucide-react"
import { ThemeProvider } from "@/components/theme-provider"
import Navbar from "./navbar"
import { Button } from "@/components/ui/button"
import { Card } from "@/components/ui/card"
import { dynamicActivate } from "@/lib/i18n"
import { $chartTime, $userSettings, $copyContent, $allSystemsById } from "@/lib/stores"
import { chartTimeData } from "@/lib/utils"
import type { ChartTimes, SystemRecord } from "@/types"
import { base, publicGet, type Config, type System } from "./api"
import "../index.css"

import { Toaster } from "@/components/ui/toaster"
const CopyDialog = lazy(() => import("@/components/copy-to-clipboard"))
const Admin = lazy(() => import("./admin"))
const Detail = lazy(() => import("./public-detail"))
export function Empty({ children }: { children: React.ReactNode }) {
	return (
		<div className="min-h-48 grid place-content-center justify-items-center gap-3 text-center text-muted-foreground">
			{children}
		</div>
	)
}
function LoadingState({ children }: { children: React.ReactNode }) {
	return (
		<div className="min-h-48 grid place-content-center justify-items-center gap-3 text-center text-muted-foreground">
			<LoaderCircle className="size-7 animate-spin opacity-70" aria-hidden="true" />
			<span>{children}</span>
		</div>
	)
}
const savedRange = localStorage.getItem("chenv-range")
try {
	const settings = JSON.parse(localStorage.getItem("chenv-chart-settings") || "null")
	if (settings) $userSettings.set({ ...$userSettings.get(), ...settings })
} catch {
	/* Ignore stale browser preferences. */
}
if (savedRange && savedRange !== "1m" && savedRange in chartTimeData) $chartTime.set(savedRange as ChartTimes)

function Dashboard() {
	const [config, setConfig] = useState<Config>()
	const [systems, setSystems] = useState<System[]>()
	const [detail, setDetail] = useState<System>()
	const [error, setError] = useState("")
	const [reload, setReload] = useState(0)
	const range = useStore($chartTime)
	const copyContent = useStore($copyContent)
	const readPath = () => window.location.pathname.slice(base.length).replace(/\/$/, "") || "/"
	const [path, setPath] = useState(readPath)
	useEffect(() => {
		const update = () => setPath(readPath())
		window.addEventListener("popstate", update)
		return () => window.removeEventListener("popstate", update)
	}, [])
	const isAdmin = path === "/chenv-admin"
	const systemId = path.startsWith("/system/") ? path.slice(8) : ""
	const selectedId = systemId || systems?.find((s) => s.id === "m9gh5qbz6fc3bw3")?.id || systems?.[0]?.id || ""
	const selected = detail?.id === selectedId ? detail : undefined
	useEffect(() => {
		localStorage.setItem("chenv-range", range)
		setDetail(undefined)
	}, [range])
	useEffect(() => {
		const controller = new AbortController()
		publicGet<Config>("config", controller.signal)
			.then((c) => {
				setConfig(c)
				document.title = c.name
				for (const icon of document.querySelectorAll<HTMLLinkElement>('link[rel="icon"],link[rel="apple-touch-icon"]'))
					icon.href = c.favicon.startsWith("/") ? base + c.favicon : c.favicon
			})
			.catch((e) => {
				if (!controller.signal.aborted) setError(e.message)
			})
		return () => controller.abort()
	}, [reload])
	useEffect(() => {
		if (isAdmin) return
		const controller = new AbortController()
		let timer: ReturnType<typeof setTimeout>
		const load = async () => {
			try {
				const rows = await publicGet<System[]>("systems", controller.signal)
				setSystems(rows)
				$allSystemsById.set(
					Object.fromEntries(
						rows.map((s) => [s.id, { ...s, host: s.host || "", port: s.port || "" }])
					) as unknown as Record<string, SystemRecord>
				)
				setError("")
			} catch (e) {
				if (!controller.signal.aborted) setError((e as Error).message)
			} finally {
				if (!controller.signal.aborted) timer = setTimeout(load, 10000)
			}
		}
		void load()
		return () => {
			controller.abort()
			clearTimeout(timer)
		}
	}, [isAdmin, reload])
	useEffect(() => {
		if (!selectedId || isAdmin) return
		const controller = new AbortController()
		let timer: ReturnType<typeof setTimeout>
		const load = async () => {
			try {
				const d = await publicGet<System>(`systems/${encodeURIComponent(selectedId)}?range=${range}`, controller.signal)
				setDetail(d)
				setError("")
			} catch (e) {
				if (!controller.signal.aborted) {
					setDetail(undefined)
					setError((e as Error).message)
				}
			} finally {
				if (!controller.signal.aborted) timer = setTimeout(load, 10000)
			}
		}
		void load()
		return () => {
			controller.abort()
			clearTimeout(timer)
		}
	}, [selectedId, range, isAdmin, reload])
	return (
		<ThemeProvider defaultTheme={config?.theme || "system"}>
			<div style={{ "--container": "1600px" } as React.CSSProperties}>
				{!isAdmin && (
					<div className="container">
						<Navbar name={config?.name} />
					</div>
				)}
				<main className={isAdmin ? "relative" : "container relative pb-8"} id="main">
					{isAdmin ? (
						<Suspense fallback={<LoadingState>加载管理后台…</LoadingState>}>
							<Admin config={config} refreshConfig={() => setReload((v) => v + 1)} />
						</Suspense>
					) : path !== "/" && !systemId ? (
						<Empty>
							页面不存在 <a href={`${base}/`}>返回首页</a>
						</Empty>
					) : (
						<>
							{error && (
								<div
									className="rounded-md border border-destructive/50 text-destructive p-4 mb-4 flex items-center justify-between gap-4"
									role="alert"
								>
									{error}
									<Button
										variant="outline"
										onClick={() => {
											setError("")
											setReload((v) => v + 1)
										}}
									>
										<RefreshCw className="size-4" />
										重试
									</Button>
								</div>
							)}
							{!systems && !error ? (
								<LoadingState>加载看板…</LoadingState>
							) : !systems?.length && !systemId ? (
								<Card>
									<Empty>
										<Server className="size-9" />
										<h1 className="text-xl">暂无公开服务器</h1>
										<p>管理员尚未公开服务器，请稍后再来。</p>
									</Empty>
								</Card>
							) : selectedId ? (
								selected ? (
									<>
										<Suspense fallback={<LoadingState>加载图表…</LoadingState>}>
											<Detail system={selected} />
										</Suspense>
									</>
								) : (
									!error && <LoadingState>加载服务器指标…</LoadingState>
								)
							) : null}
						</>
					)}
				</main>
				{!isAdmin && (
					<footer className="container pb-4 pt-1">
						<div className="mx-auto flex w-fit max-w-full items-center justify-center gap-1.5 rounded-full border bg-card/60 px-2.5 py-1 text-xs leading-none text-muted-foreground shadow-xs">
							<span>Powered by</span>
							<a
								className="inline-flex min-w-0 items-center gap-1.5 font-medium text-foreground transition-colors hover:text-primary hover:underline"
								href="https://github.com/kerntau/monitor"
								target="_blank"
								rel="noreferrer"
							>
								<GithubIcon className="size-3.5 shrink-0" aria-hidden="true" />
								<span className="truncate">kerntau/monitor</span>
							</a>
						</div>
					</footer>
				)}
			</div>
			<Toaster />
			{copyContent && (
				<Suspense>
					<CopyDialog content={copyContent} />
				</Suspense>
			)}
		</ThemeProvider>
	)
}
void dynamicActivate("zh-CN").then(() =>
	createRoot(document.getElementById("app")!).render(
		<I18nProvider i18n={i18n}>
			<Dashboard />
		</I18nProvider>
	)
)
