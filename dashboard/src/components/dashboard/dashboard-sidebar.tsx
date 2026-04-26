import { MdOutlineMonitorHeart } from 'react-icons/md'

import { ScrollArea } from '@/components/ui/scroll-area'
import { cn } from '@/lib/utils'

import { mainNav, type NavItem } from './nav-config'

type DashboardSidebarProps = {
  activeId?: string
  onNavigate?: () => void
}

function NavLink({
  item,
  active,
  onClick,
}: {
  item: NavItem
  active: boolean
  onClick?: () => void
}) {
  const Icon = item.icon
  return (
    <a
      href={item.href}
      onClick={onClick}
      className={cn(
        'group flex items-center gap-3 rounded-md px-3 py-2 text-sm font-medium transition-colors',
        active
          ? 'bg-primary/15 text-primary'
          : 'text-muted-foreground hover:bg-sidebar-accent hover:text-foreground'
      )}
    >
      <Icon
        className={cn(
          'size-5 shrink-0',
          active ? 'text-primary' : 'text-muted-foreground group-hover:text-foreground'
        )}
        aria-hidden
      />
      {item.label}
    </a>
  )
}

export function DashboardSidebar({
  activeId = 'connections',
  onNavigate,
}: DashboardSidebarProps) {
  return (
    <div className="flex h-full min-h-0 flex-col border-sidebar-border bg-sidebar">
      <div className="flex h-14 shrink-0 items-center gap-2 border-b border-sidebar-border px-4">
        <div className="flex size-9 items-center justify-center rounded-lg bg-primary/15 text-primary">
          <MdOutlineMonitorHeart className="size-5" aria-hidden />
        </div>
        <div className="min-w-0">
          <p className="truncate text-sm font-semibold tracking-tight text-foreground">
            Manitor
          </p>
          <p className="truncate text-xs text-muted-foreground">Dashboard</p>
        </div>
      </div>

      <ScrollArea className="min-h-0 flex-1 px-3 py-4">
        <nav className="space-y-1" aria-label="Main">
          {mainNav.map((item) => (
            <NavLink
              key={item.label}
              item={item}
              active={item.id === activeId}
              onClick={onNavigate}
            />
          ))}
        </nav>
      </ScrollArea>
    </div>
  )
}
