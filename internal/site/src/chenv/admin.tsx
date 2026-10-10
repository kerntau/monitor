import { useEffect, useState, type FormEvent, type ReactNode } from "react"
import { LogOut, Server, Network, Settings, KeyRound, MailIcon, LockIcon, LogInIcon, LoaderCircle } from "lucide-react"
import { admin, adminRequest, base, type AdminSystem, type Config } from "./api"

import { LoginLayout } from "@/components/login/layout"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { Checkbox } from "@/components/ui/checkbox"
import { SystemDialog } from "@/components/add-system"
import TokensPage from "@/components/routes/settings/tokens-fingerprints"
import { $publicKey, $allSystemsById, $longestSystemName } from "@/lib/stores"
import type { SystemRecord } from "@/types"
import { SettingsFrame } from "@/components/routes/settings/layout-frame"
import { SidebarNav } from "@/components/routes/settings/sidebar-nav"
import { Dialog as DialogRoot, DialogContent, DialogHeader, DialogTitle } from "@/components/ui/dialog"
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select"
import Navbar from "./navbar"
import ChenvNetwork from "./network"
import SystemsTable from "@/components/systems-table/systems-table"
import { SystemManagementContext } from "@/lib/system-management-context"

function Field({ label, children }: { label: string; children: ReactNode }) {
	return (
		<label className="grid gap-2 mb-4 text-sm">
			<span className="font-medium">{label}</span>
			{children}
		</label>
	)
}
function Check({ label, checked, onChange }: { label: string; checked: boolean; onChange: (v: boolean) => void }) {
	return (
		<label className="flex items-center gap-2 py-2 text-sm">
			<Checkbox checked={checked} onCheckedChange={(v) => onChange(v === true)} />
			{label}
		</label>
	)
}
function Dialog({ title, children, close }: { title: string; children: ReactNode; close: () => void }) {
	return (
		<DialogRoot open onOpenChange={(open) => !open && close()}>
			<DialogContent
				aria-describedby={undefined}
				className="w-[90%] max-h-[90svh] overflow-auto rounded-lg sm:max-w-lg"
			>
				<DialogHeader>
					<DialogTitle>{title}</DialogTitle>
				</DialogHeader>
				{children}
			</DialogContent>
		</DialogRoot>
	)
}

const newSystem: AdminSystem = {
	name: "",
	host: "",
	port: "45876",
	paused: false,
	displayName: "",
	public: false,
	sort: 0,
}

export default function Admin({ config, refreshConfig }: { config?: Config; refreshConfig: () => void }) {
	const [logged, setLogged] = useState(false)
	const [checking, setChecking] = useState(true)
	const [tab, setTab] = useState("systems")
	const [systems, setSystems] = useState<AdminSystem[]>([])
	const [settings, setSettings] = useState<Config>(
		config || { name: "chenv", favicon: "/static/chenv-avatar.png", theme: "system", hideAddresses: true }
	)
	const [system, setSystem] = useState<AdminSystem>()
	const [install, setInstall] = useState<{ system?: AdminSystem; key: string; token: string }>()
	const [error, setError] = useState("")
	const [message, setMessage] = useState("")
	const [busy, setBusy] = useState(false)
	useEffect(() => {
		const expired = () => {
			setLogged(false)
			setError("登录已过期，请重新登录")
			setSystem(undefined)
			setInstall(undefined)
		}
		window.addEventListener("chenv-auth-expired", expired)
		let active = true
		if (admin.authStore.isValid && admin.authStore.record?.collectionName === "_superusers") {
			admin
				.collection("_superusers")
				.authRefresh()
				.then(() => {
					if (active) setLogged(true)
				})
				.catch(() => admin.authStore.clear())
				.finally(() => {
					if (active) setChecking(false)
				})
		} else {
			admin.authStore.clear()
			setChecking(false)
		}
		return () => {
			active = false
			window.removeEventListener("chenv-auth-expired", expired)
		}
	}, [])
	const refresh = async () => {
		const [s, c, info] = await Promise.all([
			adminRequest<AdminSystem[]>("systems"),
			adminRequest<Config>("config"),
			adminRequest<{ key: string }>("agent-info"),
		])
		$publicKey.set(info.key)
		$allSystemsById.set(
			Object.fromEntries(s.map((v) => [v.id, { ...v, info: (v as unknown as SystemRecord).info || {} }])) as Record<
				string,
				SystemRecord
			>
		)
		setSystems(
			s.map((v) => ({
				...v,
				info: (v as unknown as SystemRecord).info || {},
				paused: v.status === "paused",
				displayName: v.displayName || v.name,
				public: !!v.public,
				sort: v.sort || 0,
			}))
		)
		setSettings(c)
		$longestSystemName.set(s.reduce((name, v) => (v.name.length > name.length ? v.name : name), ""))
	}
	useEffect(() => {
		if (logged) void refresh().catch((e) => setError(e.message))
	}, [logged])
	useEffect(() => {
		if (!logged) return
		let active = true
		const timer = setInterval(
			() =>
				void adminRequest<AdminSystem[]>("systems")
					.then((s) => {
						if (!active) return
						setSystems(
							s.map((v) => ({
								...v,
								info: (v as unknown as SystemRecord).info || {},
								paused: v.status === "paused",
								displayName: v.displayName || v.name,
								public: !!v.public,
								sort: v.sort || 0,
							}))
						)
						$allSystemsById.set(
							Object.fromEntries(
								s.map((v) => [v.id, { ...v, info: (v as unknown as SystemRecord).info || {} }])
							) as Record<string, SystemRecord>
						)
						$longestSystemName.set(s.reduce((name, v) => (v.name.length > name.length ? v.name : name), ""))
					})
					.catch((e) => {
						if (active) setError(e.message)
					}),
			10000
		)
		return () => {
			active = false
			clearInterval(timer)
		}
	}, [logged])
	useEffect(() => {
		if (!message) return
		const timer = setTimeout(() => setMessage(""), 5000)
		return () => clearTimeout(timer)
	}, [message])
	const run = async (action: () => Promise<void>) => {
		setBusy(true)
		setError("")
		try {
			await action()
		} catch (e) {
			setError((e as Error).message)
		} finally {
			setBusy(false)
		}
	}
	const logout = () => {
		admin.authStore.clear()
		setLogged(false)
		setSystem(undefined)
		setInstall(undefined)
		setError("")
	}
	const login = (e: FormEvent<HTMLFormElement>) => {
		e.preventDefault()
		const f = new FormData(e.currentTarget)
		void run(async () => {
			await admin.collection("_superusers").authWithPassword(String(f.get("email")), String(f.get("password")))
			setLogged(true)
		})
	}
	const saveSystem = (e: FormEvent) => {
		e.preventDefault()
		if (!system) return
		void run(async () => {
			await adminRequest(`systems${system.id ? "/" + system.id : ""}`, system.id ? "PUT" : "POST", system)
			setSystem(undefined)
			await refresh()
			setMessage("服务器已保存")
		})
	}

	const openSystem = async (s?: AdminSystem) => {
		const credentials = await adminRequest<{ key: string; token: string }>(s ? `systems/${s.id}/agent` : "agent-info")
		setInstall({ system: s, ...credentials })
	}
	const saveConnection = async (data: Record<string, FormDataEntryValue>) => {
		const original = install?.system || { ...newSystem, sort: systems.length }
		await adminRequest(`systems${original.id ? "/" + original.id : ""}`, original.id ? "PUT" : "POST", {
			...original,
			name: String(data.name),
			host: String(data.host),
			port: String(data.port || ""),
			token: install?.token,
		})
		await refresh()
		setMessage("服务器已保存")
	}

	const password = (e: FormEvent<HTMLFormElement>) => {
		e.preventDefault()
		const f = new FormData(e.currentTarget)
		if (f.get("password") !== f.get("confirmation")) {
			setError("两次新密码不一致")
			return
		}
		void run(async () => {
			await adminRequest("password", "POST", { current: f.get("current"), password: f.get("password") })
			logout()
			setMessage("密码已修改，请使用新密码登录")
		})
	}
	const alerts = (
		<>
			{error && (
				<div className="rounded-md border border-destructive/40 text-destructive p-3 mb-4 text-sm" role="alert">
					{error}
				</div>
			)}
			{message && (
				<div className="rounded-md border p-3 mb-4 text-sm" role="status">
					{message}
				</div>
			)}
		</>
	)
	const sections = [
		["systems", "服务器", Server],
		["monitors", "网络探测", Network],
		["tokens", "令牌与指纹", KeyRound],
		["settings", "展示设置", Settings],
		["password", "修改密码", KeyRound],
	] as const
	if (checking)
		return <div className="min-h-svh grid place-items-center text-sm text-muted-foreground">检查管理员登录状态…</div>
	if (!logged)
		return (
			<LoginLayout title="管理员登录" subtitle="请登录您的管理员账户">
				{alerts}
				<div className="grid gap-6">
					<form onSubmit={login}>
						<div className="grid gap-2.5">
							<div className="grid gap-1 relative">
								<MailIcon className="absolute left-3 top-3 h-4 w-4 text-muted-foreground" />
								<Label className="sr-only" htmlFor="admin-email">
									邮箱
								</Label>
								<Input
									id="admin-email"
									name="email"
									type="email"
									placeholder="name@example.com"
									autoComplete="username"
									autoCapitalize="none"
									required
									disabled={busy}
									className="ps-9"
								/>
							</div>
							<div className="grid gap-1 relative">
								<LockIcon className="absolute left-3 top-3 h-4 w-4 text-muted-foreground" />
								<Label className="sr-only" htmlFor="admin-password">
									密码
								</Label>
								<Input
									id="admin-password"
									name="password"
									type="password"
									placeholder="密码"
									autoComplete="current-password"
									required
									disabled={busy}
									className="ps-9"
								/>
							</div>
							<Button disabled={busy}>
								{busy ? <LoaderCircle className="size-4 animate-spin" /> : <LogInIcon className="size-4" />}
								{busy ? "登录中…" : "登录"}
							</Button>
						</div>
					</form>
					<a className="text-center text-sm text-muted-foreground hover:text-foreground" href={`${base}/`}>
						公开看板
					</a>
				</div>
			</LoginLayout>
		)
	return (
		<div className="container pb-8">
			<Navbar
				name={config?.name}
				onAddSystem={() => {
					setTab("systems")
					void run(() => openSystem())
				}}
			/>
			<SettingsFrame
				title="管理后台"
				description={admin.authStore.record?.email}
				toolbar={
					<Button variant="ghost" onClick={logout}>
						<LogOut className="size-4 me-2" />
						退出
					</Button>
				}
				sidebar={
					<>
						<SidebarNav
							aria-label="管理功能"
							value={tab}
							onValueChange={(v) => {
								setTab(v)
								setError("")
							}}
							items={sections.map(([href, title, icon]) => ({ href, title, icon }))}
						/>
						<a className="link text-sm" href={`${base}/`}>
							公开看板
						</a>
					</>
				}
			>
				{alerts}
				{tab === "systems" && (
					<SystemManagementContext.Provider
						value={{
							edit: (record) => void run(() => openSystem(systems.find((s) => s.id === record.id))),
							display: (record) => setSystem(systems.find((s) => s.id === record.id)),
							pause: (record) =>
								void run(async () => {
									const s = systems.find((s) => s.id === record.id)!
									await adminRequest("systems/" + s.id, "PUT", { ...s, paused: !s.paused })
									await refresh()
								}),
							remove: (record) =>
								void run(async () => {
									await adminRequest("systems/" + record.id, "DELETE")
									await refresh()
								}),
						}}
					>
						<SystemsTable records={systems as unknown as SystemRecord[]} management />
					</SystemManagementContext.Provider>
				)}
				{tab === "monitors" && <ChenvNetwork adminMode />}
				{tab === "tokens" && <TokensPage client={admin} universalEndpoint="/api/chenv/admin/universal-token" />}
				{tab === "settings" && (
					<section className="max-w-2xl">
						<h2 className="text-xl font-medium mb-2">展示设置</h2>
						<form
							onSubmit={(e) => {
								e.preventDefault()
								void run(async () => {
									await adminRequest("config", "PUT", settings)
									refreshConfig()
									setMessage("展示设置已保存")
								})
							}}
						>
							<Field label="站点名称">
								<Input
									required
									maxLength={80}
									value={settings.name}
									onChange={(e) => setSettings({ ...settings, name: e.target.value })}
								/>
							</Field>
							<Field label="Favicon 地址">
								<Input
									required
									value={settings.favicon}
									onChange={(e) => setSettings({ ...settings, favicon: e.target.value })}
								/>
								<small>HTTPS 图片地址或站内路径，如 /static/chenv-avatar.png</small>
							</Field>
							<Field label="默认主题">
								<Select
									value={settings.theme}
									onValueChange={(v) => setSettings({ ...settings, theme: v as Config["theme"] })}
								>
									<SelectTrigger aria-label="默认主题">
										<SelectValue />
									</SelectTrigger>
									<SelectContent>
										<SelectItem value="system">跟随系统</SelectItem>
										<SelectItem value="light">浅色</SelectItem>
										<SelectItem value="dark">深色</SelectItem>
									</SelectContent>
								</Select>
							</Field>
							<Check
								label="公开内容隐藏 IP 和连接地址"
								checked={settings.hideAddresses}
								onChange={(v) => setSettings({ ...settings, hideAddresses: v })}
							/>
							<p className="text-sm text-muted-foreground">
								在服务器端过滤地址，管理员仍可查看原始连接信息。访客自己选择的主题优先于默认主题。
							</p>
							<Button variant="default" disabled={busy}>
								{busy ? "保存中…" : "保存设置"}
							</Button>
						</form>
					</section>
				)}
				{tab === "password" && (
					<section className="max-w-2xl">
						<h2 className="text-xl font-medium mb-2">修改管理员密码</h2>
						<form onSubmit={password}>
							<Field label="当前密码">
								<Input name="current" type="password" autoComplete="current-password" required />
							</Field>
							<Field label="新密码">
								<Input name="password" type="password" autoComplete="new-password" required minLength={12} />
								<small>至少 12 位，保存后需要重新登录。</small>
							</Field>
							<Field label="确认新密码">
								<Input name="confirmation" type="password" autoComplete="new-password" required minLength={12} />
							</Field>
							<Button variant="default" disabled={busy}>
								修改密码
							</Button>
						</form>
					</section>
				)}
			</SettingsFrame>
			{install && (
				<DialogRoot open onOpenChange={(open) => !open && setInstall(undefined)}>
					<SystemDialog
						system={install.system as unknown as SystemRecord}
						setOpen={(open) => !open && setInstall(undefined)}
						adapter={{
							key: install.key,
							token: install.token,
							onSubmit: saveConnection,
							onTokensClick: () => setTab("tokens"),
						}}
					/>
				</DialogRoot>
			)}
			{system && (
				<Dialog title="服务器展示设置" close={() => setSystem(undefined)}>
					<form onSubmit={saveSystem}>
						<Field label="公开显示名称">
							<Input
								maxLength={120}
								value={system.displayName}
								onChange={(e) => setSystem({ ...system, displayName: e.target.value })}
							/>
						</Field>
						<Field label="显示排序">
							<Input
								type="number"
								step={1}
								value={system.sort}
								onChange={(e) => setSystem({ ...system, sort: Number(e.target.value) })}
							/>
						</Field>
						<Check
							label="公开这台服务器"
							checked={system.public}
							onChange={(v) => setSystem({ ...system, public: v })}
						/>
						<Check label="暂停采集" checked={system.paused} onChange={(v) => setSystem({ ...system, paused: v })} />
						<Button disabled={busy}>保存服务器</Button>
					</form>
				</Dialog>
			)}
		</div>
	)
}
