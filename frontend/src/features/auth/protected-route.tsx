import { Navigate, Outlet, useLocation, useParams } from 'react-router-dom'

import { ErrorState } from '@/components/common/error-state'
import { RouteFallback } from '@/components/common/skeletons'

import { useSession } from './session-provider'
import { WorkspaceProvider } from './workspace-context'

// Guard for authenticated `/:workspaceSlug` subtrees. Rendered as a layout
// route, so it yields an <Outlet /> (wrapped in the workspace context) once the
// session is valid and the URL slug resolves to one of the user's workspaces.
export function ProtectedRoute() {
  const { status, error, idBySlug, workspaceById, defaultWorkspace } =
    useSession()
  const location = useLocation()
  const { workspaceSlug } = useParams()

  if (error) {
    return (
      <div className="flex min-h-svh w-full items-center justify-center p-6">
        <div className="w-full max-w-md">
          <ErrorState error={error} />
        </div>
      </div>
    )
  }

  if (status === 'loading') return <RouteFallback />

  if (status === 'unauthenticated') {
    // Preserve the attempted URL so the login flow (Phase 2.1) can return here.
    const next = encodeURIComponent(location.pathname + location.search)
    return <Navigate to={`/login?next=${next}`} replace />
  }

  // Authenticated: resolve the URL slug to a workspace id.
  const id = workspaceSlug ? idBySlug.get(workspaceSlug) : undefined
  const workspace = id ? workspaceById.get(id) : undefined

  if (!workspace?.id || !workspace.slug) {
    // Unknown/removed slug -> send the user to their default workspace.
    // Onboarding guarantees at least one workspace, so this is friendlier than
    // a 404 for stale or shared-but-inaccessible links.
    if (defaultWorkspace?.slug) {
      return <Navigate to={`/${defaultWorkspace.slug}`} replace />
    }
    return <Navigate to="/login" replace />
  }

  return (
    <WorkspaceProvider
      value={{
        id: workspace.id,
        slug: workspace.slug,
        role: workspace.role ?? '',
      }}
    >
      <Outlet />
    </WorkspaceProvider>
  )
}
