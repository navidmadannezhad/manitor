import { HiMenuAlt2 } from 'react-icons/hi'
import { IoNotificationsOutline, IoSearchOutline } from 'react-icons/io5'

import { Button } from '@/components/ui/button'
import { Separator } from '@/components/ui/separator'
import {
  Sheet,
  SheetContent,
  SheetTrigger,
} from '@/components/ui/sheet'

import { DashboardSidebar } from './dashboard-sidebar'
import { mainNav } from './nav-config'

type DashboardHeaderProps = {
  activeId?: string
  mobileNavOpen: boolean
  onMobileNavOpenChange: (open: boolean) => void
}

export function DashboardHeader({
  activeId = 'connections',
  mobileNavOpen,
  onMobileNavOpenChange,
}: DashboardHeaderProps) {
  const title =
    mainNav.find((item) => item.id === activeId)?.label ?? 'Dashboard'

  return (
    <header className="sticky top-0 z-40 flex h-14 shrink-0 items-center gap-2 border-b border-border bg-header/95 px-4 backdrop-blur supports-[backdrop-filter]:bg-header/80">
      <Sheet open={mobileNavOpen} onOpenChange={onMobileNavOpenChange}>
        <SheetTrigger asChild>
          <Button
            variant="ghost"
            size="icon"
            className="lg:hidden"
            aria-label="Open navigation"
          >
            <HiMenuAlt2 className="size-5" />
          </Button>
        </SheetTrigger>
        <SheetContent
          side="left"
          className="w-64 border-sidebar-border bg-sidebar p-0 lg:hidden"
        >
          <DashboardSidebar
            activeId={activeId}
            onNavigate={() => onMobileNavOpenChange(false)}
          />
        </SheetContent>
      </Sheet>

      <div className="flex min-w-0 flex-1 items-center gap-2">
        <h1 className="truncate text-sm font-medium text-foreground sm:text-base">
          {title}
        </h1>
      </div>

      <div className="flex items-center gap-1 sm:gap-2">
        <Button
          variant="ghost"
          size="icon"
          className="text-muted-foreground"
          aria-label="Search"
        >
          <IoSearchOutline className="size-5" />
        </Button>
        <Button
          variant="ghost"
          size="icon"
          className="relative text-muted-foreground"
          aria-label="Notifications"
        >
          <IoNotificationsOutline className="size-5" />
          <span className="absolute right-1.5 top-1.5 size-2 rounded-full bg-primary ring-2 ring-header" />
        </Button>
        <Separator orientation="vertical" className="hidden h-6 sm:block" />
        <div
          className="hidden size-8 shrink-0 rounded-full bg-primary/20 ring-1 ring-primary/40 sm:block"
          aria-hidden
        />
      </div>
    </header>
  )
}
