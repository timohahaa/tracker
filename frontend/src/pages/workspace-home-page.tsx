import { useParams } from 'react-router-dom'

// Placeholder for a protected workspace route. No domain data yet — the
// app-shell (sidebar/topbar) lands in Phase 1.4 and workspace resolution in 1.3.
export default function WorkspaceHomePage() {
  const { workspaceSlug } = useParams()

  return (
    <div className="flex min-h-svh flex-col items-center justify-center gap-2 p-6 text-center">
      <h1 className="text-2xl font-semibold">Workspace home</h1>
      <p className="text-sm text-muted-foreground">
        slug: <code className="font-mono">{workspaceSlug}</code>
      </p>
    </div>
  )
}
