import { createContext, useContext } from "react"
import type { SystemRecord } from "@/types"
export type SystemManagementAdapter = {
	edit: (system: SystemRecord) => void
	display: (system: SystemRecord) => void
	pause: (system: SystemRecord) => void
	remove: (system: SystemRecord) => void
}
export const SystemManagementContext = createContext<SystemManagementAdapter | null>(null)
export const useSystemManagement = () => useContext(SystemManagementContext)
