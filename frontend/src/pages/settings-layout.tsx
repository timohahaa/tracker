import { NavLink, Outlet } from 'react-router-dom'

import { useCurrentWorkspace } from '@/features/auth/workspace-context'
import { cn } from '@/lib/utils'

// Settings sub-sections. Kept as a single source of truth so new areas can slot
// in additional tabs here without touching the route tree beyond registering the
// child routes.
//
// `to` is appended to `/:slug/settings`; an empty string is the index (General).
// `end` marks index/leaf links so a parent tab isn't highlighted on a child path.
const SETTINGS_TABS: { label: string; to: string; end: boolean }[] = [
  { label: 'General', to: '', end: true },
  { label: 'Members', to: 'members', end: true },
  { label: 'Labels', to: 'labels', end: true },
  { label: 'Integrations', to: 'integrations', end: true },
  { label: 'API tokens', to: 'tokens', end: true },
]

// Layout route for the workspace Settings area: a shared header + a horizontal
// tab bar over the routed <Outlet/>. Reachable from the sidebar Settings link.
export default function SettingsLayout() {
  const { slug } = useCurrentWorkspace()

  return (
    <div className="mx-auto w-full max-w-3xl px-6 py-8">
      <header className="mb-6">
        <h1 className="text-2xl font-semibold tracking-tight">Settings</h1>
        <p className="text-sm text-muted-foreground">
          Manage your workspace and members.
        </p>
      </header>

      <nav
        className="mb-6 flex gap-1 border-b"
        aria-label="Settings sections"
      >
        {SETTINGS_TABS.map(({ label, to, end }) => (
          <NavLink
            key={label}
            to={to ? `/${slug}/settings/${to}` : `/${slug}/settings`}
            end={end}
            className={({ isActive }) =>
              cn(
                '-mb-px border-b-2 px-3 py-2 text-sm font-medium transition-colors',
                'focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-ring',
                isActive
                  ? 'border-foreground text-foreground'
                  : 'border-transparent text-muted-foreground hover:text-foreground',
              )
            }
          >
            {label}
          </NavLink>
        ))}
      </nav>

      <Outlet />
    </div>
  )
}
