import PocketBase, { LocalAuthStore } from "pocketbase"

export const base = (globalThis.BESZEL?.BASE_PATH || "/").replace(/\/$/, "")
export const admin = new PocketBase(base || "/", new LocalAuthStore("chenv-admin-auth"))
admin.autoCancellation(false)
admin.afterSend = (response, data) => {
	if (response.status === 401 || response.status === 403) {
		admin.authStore.clear()
		window.dispatchEvent(new Event("chenv-auth-expired"))
	}
	return data
}
export type Config = { name: string; favicon: string; theme: "light" | "dark" | "system"; hideAddresses: boolean }
export type Stats = {
	cpu?: number
	mp?: number
	dp?: number
	m?: number
	mu?: number
	d?: number
	du?: number
	la?: number[]
	b?: number[]
	ns?: number
	nr?: number
}
export type Sample = { created: string | number; stats: Stats }
export type Probe = {
	id: string
	name: string
	protocol: string
	enabled: boolean
	res: number
	loss1h: number
	updated: string
	target?: string
	port?: number
	history: { created: number; latency: number; loss: number }[]
}
export type System = {
	id: string
	name: string
	status: string
	updated: string
	host?: string
	port?: string
	info: Stats & { u?: number }
	hardware?: {
		hostname?: string
		kernel?: string
		cpu?: string
		cores?: number
		threads?: number
		memory?: number
		os_name?: string
		arch?: string
	}
	history?: Sample[]
	monitors?: Probe[]
}
export type AdminSystem = {
	id?: string
	name: string
	host: string
	port: string
	status?: string
	paused: boolean
	displayName: string
	public: boolean
	sort: number
}
export type AdminProbe = {
	id?: string
	name: string
	system: string
	target: string
	protocol: string
	port: number
	server: string
	interval: number
	enabled: boolean
	public: boolean
}
export async function publicGet<T>(path: string, signal?: AbortSignal): Promise<T> {
	const r = await fetch(`${base}/api/chenv/public/${path}`, { signal, credentials: "omit" })
	if (!r.ok) throw new Error(r.status === 404 ? "服务器未公开或已被删除" : "加载失败，请稍后重试")
	return r.json()
}
export async function adminRequest<T>(path: string, method = "GET", body?: unknown): Promise<T> {
	try {
		return await admin.send<T>(`/api/chenv/admin/${path}`, { method, body })
	} catch (error) {
		const e = error as { status?: number; message?: string; response?: { message?: string } }
		if (e.status === 401 || e.status === 403) {
			admin.authStore.clear()
			window.dispatchEvent(new Event("chenv-auth-expired"))
		}
		throw new Error(e.response?.message || e.message || "操作失败，请重试")
	}
}
