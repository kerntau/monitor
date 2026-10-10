import type { ReactNode } from 'react'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Separator } from '@/components/ui/separator'

/** Original settings layout, with content and navigation supplied by the caller. */
export function SettingsFrame({title, description, sidebar, children, toolbar}: {title: ReactNode; description: ReactNode; sidebar: ReactNode; children: ReactNode; toolbar?: ReactNode}) {
 return <Card className="pt-5 px-4 pb-8 min-h-96 mb-14 sm:pt-6 sm:px-7">
  <CardHeader className="p-0"><div className="flex items-center justify-between gap-4"><div><CardTitle className="mb-1">{title}</CardTitle><CardDescription>{description}</CardDescription></div>{toolbar}</div></CardHeader>
  <CardContent className="p-0"><Separator className="hidden md:block my-5"/>
   <div className="flex flex-col gap-3.5 md:flex-row md:gap-5 lg:gap-12"><aside className="md:max-w-52 min-w-40">{sidebar}</aside><div className="flex-1 min-w-0">{children}</div></div>
  </CardContent>
 </Card>
}
