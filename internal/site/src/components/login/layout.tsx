import type { CSSProperties, ReactNode } from 'react'
import { Logo } from '../logo'
import { ModeToggle } from '../mode-toggle'
import { useTheme } from '../theme-provider'

/** Shared original Beszel sign-in layout; authentication is supplied by the caller. */
export function LoginLayout({subtitle, children, title = 'chenv'}: {subtitle: string; children: ReactNode; title?: string}) {
 const {resolvedTheme} = useTheme()
 return <div className="min-h-svh grid items-center py-12">
  <div className="grid gap-5 w-full px-4 mx-auto" style={{maxWidth:'21.5em','--border':resolvedTheme === 'light' ? 'hsl(30, 8%, 70%)' : 'hsl(220, 3%, 25%)'} as CSSProperties}>
   <div className="absolute top-3 right-3"><ModeToggle/></div>
   <div className="text-center"><h1 className="mb-3"><Logo className="h-7 fill-foreground mx-auto"/><span className="sr-only">{title}</span></h1><p className="text-sm text-muted-foreground">{subtitle}</p></div>
   {children}
  </div>
 </div>
}
