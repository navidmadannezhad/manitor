import type { IconType } from 'react-icons/lib'
import { IoLinkOutline } from 'react-icons/io5'

export type NavItem = {
  id: string
  label: string
  href: string
  icon: IconType
}

export const mainNav: NavItem[] = [
  {
    id: 'connections',
    label: 'Connections',
    href: '#connections',
    icon: IoLinkOutline,
  },
]
