import { useState, type ReactNode } from 'react'

import { ConnectionsPage } from './connections-page'
import { DashboardHeader } from './dashboard-header'
import { DashboardSidebar } from './dashboard-sidebar'

type DashboardLayoutProps = {
  children?: ReactNode
  activeNavId?: string
}

export function DashboardLayout({
  children,
  activeNavId = 'connections',
}: DashboardLayoutProps) {
  const [mobileNavOpen, setMobileNavOpen] = useState(false)

  return (
    <div className="flex min-h-svh w-full bg-background text-foreground">
      <aside className="sticky top-0 hidden h-svh w-64 shrink-0 border-r border-sidebar-border lg:block">
        <DashboardSidebar activeId={activeNavId} />
      </aside>

      <div className="flex min-h-svh min-w-0 flex-1 flex-col">
        <DashboardHeader
          activeId={activeNavId}
          mobileNavOpen={mobileNavOpen}
          onMobileNavOpenChange={setMobileNavOpen}
        />

        <main className="flex-1 overflow-auto p-4 sm:p-6">
          {children ?? <ConnectionsPage />}
        </main>
      </div>
    </div>
  )
}
