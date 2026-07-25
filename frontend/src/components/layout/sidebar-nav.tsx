import {
  FolderKanban,
  Home,
  ListTodo,
  Settings,
  Users,
} from 'lucide-react'
import { NavLink } from 'react-router-dom'

import { useCurrentWorkspace } from '@/features/auth/workspace-context'
import { cn } from '@/lib/utils'

interface NavItem {
  label: string
  // Path segment appended to the workspace slug. Empty string -> the workspace
  // index (home).
  to: string
  icon: React.ComponentType<{ className?: string }>
  // NavLink `end` so the index link isn't marked active for nested routes.
  end?: boolean
}

// Static navigation placeholders (Phase 1.4). The target pages arrive in Phase
// 2.x; until then these resolve to the in-shell NotFound child. No data
// fetching here.
const NAV_ITEMS: NavItem[] = [
  { label: 'Home', to: '', icon: Home, end: true },
  { label: 'Issues', to: 'issues', icon: ListTodo },
  { label: 'Projects', to: 'projects', icon: FolderKanban },
  { label: 'Teams', to: 'teams', icon: Users },
  { label: 'Settings', to: 'settings', icon: Settings },
]

interface SidebarNavProps {
  // Invoked when a link is activated, so the mobile drawer can close itself.
  onNavigate?: () => void
}

export function SidebarNav({ onNavigate }: SidebarNavProps) {
  const { slug } = useCurrentWorkspace()

  return (
    <nav className="flex flex-col gap-1" aria-label="Primary">
      {NAV_ITEMS.map(({ label, to, icon: Icon, end }) => {
        const href = to ? `/${slug}/${to}` : `/${slug}`
        return (
          <NavLink
            key={label}
            to={href}
            end={end}
            onClick={onNavigate}
            className={({ isActive }) =>
              cn(
                'flex items-center gap-2.5 rounded-md px-3 py-2 text-sm font-medium transition-colors',
                'hover:bg-accent hover:text-accent-foreground',
                'focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-ring',
                isActive
                  ? 'bg-accent text-accent-foreground'
                  : 'text-muted-foreground',
              )
            }
          >
            <Icon className="h-4 w-4 shrink-0" />
            <span className="truncate">{label}</span>
          </NavLink>
        )
      })}
    </nav>
  )
}
