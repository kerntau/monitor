import { useStore } from "@nanostores/react"
import type React from "react"
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select"
import { Separator } from "@/components/ui/separator"
import { isAdmin, isReadOnlyUser } from "@/lib/api"
import { cn } from "@/lib/utils"
import { $router, Link, navigate } from "../../router"
import { buttonVariants } from "../../ui/button"

interface SidebarNavProps extends React.HTMLAttributes<HTMLElement> {
	value?: string
	onValueChange?: (value: string) => void
	items: {
		href: string
		title: string
		icon?: React.FC<React.SVGProps<SVGSVGElement>>
		admin?: boolean
		noReadOnly?: boolean
		preload?: () => Promise<{ default: React.ComponentType<any> }>
	}[]
}

export function SidebarNav({ className, items, value, onValueChange, ...props }: SidebarNavProps) {
	const page = useStore($router)

	return (
		<>
			{/* Mobile View */}
			<div className="md:hidden">
				<Select onValueChange={onValueChange ?? navigate} value={value ?? page?.path}>
					<SelectTrigger className="w-full my-3.5" aria-label={props["aria-label"]}>
						<SelectValue placeholder="Select page" />
					</SelectTrigger>
					<SelectContent>
						{items.map((item) => {
							if (item.admin && !isAdmin()) return null
							return (
								<SelectItem key={item.href} value={item.href}>
									<span className="flex items-center gap-2 truncate">
										{item.icon && <item.icon className="size-4" />}
										<span className="truncate">{item.title}</span>
									</span>
								</SelectItem>
							)
						})}
					</SelectContent>
				</Select>
				<Separator />
			</div>

			{/* Desktop View */}
			<nav className={cn("hidden md:grid gap-1 sticky top-6", className)} {...props}>
				{items.map((item) => {
					if ((item.admin && !isAdmin()) || (item.noReadOnly && isReadOnlyUser())) {
						return null
					}
					if (onValueChange) return <button type="button" key={item.href} onClick={() => onValueChange(item.href)} className={cn(buttonVariants({ variant: "ghost" }), "flex items-center gap-3 justify-start truncate duration-50", value === item.href ? "bg-muted hover:bg-accent/70" : "hover:bg-accent/50")}>{item.icon && <item.icon className="size-4 shrink-0"/>}<span className="truncate">{item.title}</span></button>
					return (
						<Link
							onMouseEnter={() => item.preload?.()}
							key={item.href}
							href={item.href}
							className={cn(
								buttonVariants({ variant: "ghost" }),
								"flex items-center gap-3 justify-start truncate duration-50",
								page?.path === item.href ? "bg-muted hover:bg-accent/70" : "hover:bg-accent/50"
							)}
						>
							{item.icon && <item.icon className="size-4 shrink-0" />}
							<span className="truncate">{item.title}</span>
						</Link>
					)
				})}
			</nav>
		</>
	)
}
