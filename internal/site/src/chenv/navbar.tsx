import { useRef } from "react"
import { Logo } from "@/components/logo"
import { ModeToggle } from "@/components/mode-toggle"
import { Button } from "@/components/ui/button"
import { PlusIcon } from "lucide-react"
import { NavbarFrame } from "@/components/navbar-frame"
import { base } from "./api"

export default function Navbar({ name = "chenv", onAddSystem }: { name?: string; onAddSystem?: () => void }) {
	const clicks = useRef({ count: 0, last: 0 })
	const secretClick = () => {
		const now = Date.now()
		clicks.current.count = now - clicks.current.last <= 3000 ? clicks.current.count + 1 : 1
		clicks.current.last = now
		if (clicks.current.count >= 5) {
			window.location.assign(`${base}/chenv-admin`)
			return
		}
		if (window.location.pathname !== `${base}/`) {
			window.history.pushState({}, "", `${base}/`)
			window.dispatchEvent(new PopStateEvent("popstate"))
		}
	}
	return (
		<NavbarFrame>
			<button onClick={secretClick} aria-label={name} className="p-2 ps-0 me-3 group touch-manipulation">
				{name === "chenv" ? (
					<Logo className="h-[1.2rem] md:h-5 fill-foreground" />
				) : (
					<span className="text-xl font-bold">{name}</span>
				)}
			</button>
			<div className="ms-auto flex items-center gap-2">
				<ModeToggle />
				{onAddSystem && (
					<Button variant="outline" onClick={onAddSystem}>
						<PlusIcon className="size-4" />
						新增主机
					</Button>
				)}
			</div>
		</NavbarFrame>
	)
}
