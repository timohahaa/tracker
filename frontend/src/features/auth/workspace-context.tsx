import { createContext, useContext } from 'react'

// The active workspace for the current `/:workspaceSlug` subtree. Provided by
// ProtectedRoute after resolving the URL slug to a workspace, and consumed by
// data hooks/pages that need the UUID `{workspaceID}` (all workspace-scoped
// backend endpoints take the id, never the slug).
export interface CurrentWorkspace {
  id: string
  slug: string
  role: string
}

const WorkspaceContext = createContext<CurrentWorkspace | null>(null)

export function WorkspaceProvider({
  value,
  children,
}: {
  value: CurrentWorkspace
  children: React.ReactNode
}) {
  return (
    <WorkspaceContext.Provider value={value}>
      {children}
    </WorkspaceContext.Provider>
  )
}

export function useCurrentWorkspace(): CurrentWorkspace {
  const ctx = useContext(WorkspaceContext)
  if (!ctx) {
    throw new Error(
      'useCurrentWorkspace must be used within a WorkspaceProvider (a /:workspaceSlug route)',
    )
  }
  return ctx
}
