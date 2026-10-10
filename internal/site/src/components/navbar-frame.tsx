import type { ReactNode } from 'react'

/** The original Beszel navigation container, shared by both entry points. */
export function NavbarFrame({children}: {children: ReactNode}) {
 return <div className="flex items-center h-14 md:h-16 bg-card px-4 pe-3 sm:px-6 border border-border/60 bt-0 rounded-md my-4">{children}</div>
}
