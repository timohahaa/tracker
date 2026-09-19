import { FolderKanban, ListTodo, Settings, Users } from 'lucide-react'
import { Link } from 'react-router-dom'

import { useCurrentUser, useWorkspaces } from '@/features/auth/session-provider'
import { useCurrentWorkspace } from '@/features/auth/workspace-context'

// Quick-jump destinations mirroring the sidebar. `to` is appended to the
// workspace slug.
const SHORTCUTS: {
  label: string
  description: string
  to: string
  icon: React.ComponentType<{ className?: string }>
}[] = [
  { label: 'Issues', description: 'Track and triage work', to: 'issues', icon: ListTodo },
  { label: 'Projects', description: 'Group work into projects', to: 'projects', icon: FolderKanban },
  { label: 'Teams', description: 'Teams, statuses and members', to: 'teams', icon: Users },
  { label: 'Settings', description: 'Workspace and members', to: 'settings', icon: Settings },
]

// Minimal workspace landing: a greeting plus quick links into the main sections.
// Kept intentionally light — richer overviews can arrive with later slices.
export default function WorkspaceHomePage() {
  const { slug, id } = useCurrentWorkspace()
  const user = useCurrentUser()
  const workspaces = useWorkspaces()

  const workspaceName = workspaces.find((ws) => ws.id === id)?.name ?? slug
  const firstName = user?.display_name?.trim().split(/\s+/)[0]

  return (
    <div className="mx-auto w-full max-w-4xl px-6 py-8">
      <header className="mb-8">
        <p className="text-sm text-muted-foreground">
          {firstName ? `Welcome back, ${firstName}` : 'Welcome back'}
        </p>
        <h1 className="text-2xl font-semibold tracking-tight">
          {workspaceName}
        </h1>
      </header>

      <div className="grid gap-3 sm:grid-cols-2">
        {SHORTCUTS.map(({ label, description, to, icon: Icon }) => (
          <Link
            key={label}
            to={`/${slug}/${to}`}
            className="flex items-start gap-3 rounded-lg border p-4 transition-colors hover:bg-accent hover:text-accent-foreground focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-ring"
          >
            <Icon className="mt-0.5 h-5 w-5 shrink-0 text-muted-foreground" />
            <div className="space-y-0.5">
              <p className="text-sm font-medium">{label}</p>
              <p className="text-xs text-muted-foreground">{description}</p>
            </div>
          </Link>
        ))}
      </div>
    </div>
  )
}
