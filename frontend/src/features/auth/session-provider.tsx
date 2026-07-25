import { createContext, useContext, useMemo } from 'react'

import { AppError } from '@/api/client'

import { type CurrentUser, useCurrentUserQuery } from './use-current-user'
import { useWorkspacesQuery,type Workspace } from './use-workspaces'

export type SessionStatus = 'loading' | 'authenticated' | 'unauthenticated'

interface SessionContextValue {
  status: SessionStatus
  // Non-401 bootstrap failure (e.g. server/network error). When set, consumers
  // render an error state instead of trusting `status`.
  error: AppError | null
  user: CurrentUser | null
  workspaces: Workspace[]
  // slug -> id and id -> slug, for client-side resolution of the URL slug.
  idBySlug: Map<string, string>
  slugById: Map<string, string>
  // Full workspace lookup by id (role, name, ...).
  workspaceById: Map<string, Workspace>
  // me.settings.current_workspace_id when valid, otherwise the first workspace.
  defaultWorkspace: Workspace | null
}

const SessionContext = createContext<SessionContextValue | null>(null)

function useSessionContext(): SessionContextValue {
  const ctx = useContext(SessionContext)
  if (!ctx) {
    throw new Error('useSession must be used within a SessionProvider')
  }
  return ctx
}

// Single source of truth for auth-bootstrap state. Fetches the current user
// (session-cookie resume) and, once that succeeds, the workspace list; derives
// the coarse `status` used to gate routing.
export function SessionProvider({ children }: { children: React.ReactNode }) {
  const meQuery = useCurrentUserQuery()
  const workspacesQuery = useWorkspacesQuery({ enabled: !!meQuery.data })

  const value = useMemo<SessionContextValue>(() => {
    const user = meQuery.data ?? null
    const workspaces = workspacesQuery.data?.data ?? []

    const idBySlug = new Map<string, string>()
    const slugById = new Map<string, string>()
    const workspaceById = new Map<string, Workspace>()
    for (const ws of workspaces) {
      if (ws.id && ws.slug) {
        idBySlug.set(ws.slug, ws.id)
        slugById.set(ws.id, ws.slug)
      }
      if (ws.id) workspaceById.set(ws.id, ws)
    }

    const preferredId = user?.settings?.current_workspace_id
    const defaultWorkspace =
      (preferredId ? workspaceById.get(preferredId) : undefined) ??
      workspaces[0] ??
      null

    // Derive status. A 401 on `me` means no/expired session -> unauthenticated;
    // any other bootstrap error surfaces via `error`.
    let status: SessionStatus = 'loading'
    let error: AppError | null = null

    if (meQuery.isError) {
      const e = meQuery.error
      if (e instanceof AppError && e.code === 'unauthorized') {
        status = 'unauthenticated'
      } else {
        error = e instanceof AppError ? e : null
      }
    } else if (user) {
      if (workspacesQuery.isError) {
        error =
          workspacesQuery.error instanceof AppError
            ? workspacesQuery.error
            : null
      } else if (workspacesQuery.data) {
        status = 'authenticated'
      }
    }

    return {
      status,
      error,
      user,
      workspaces,
      idBySlug,
      slugById,
      workspaceById,
      defaultWorkspace,
    }
  }, [
    meQuery.data,
    meQuery.isError,
    meQuery.error,
    workspacesQuery.data,
    workspacesQuery.isError,
    workspacesQuery.error,
  ])

  return (
    <SessionContext.Provider value={value}>{children}</SessionContext.Provider>
  )
}

// Full bootstrap state (status/error + resolvers).
export function useSession(): SessionContextValue {
  return useSessionContext()
}

// Current user. Only meaningful inside an authenticated subtree.
export function useCurrentUser(): CurrentUser | null {
  return useSessionContext().user
}

// The user's workspace list.
export function useWorkspaces(): Workspace[] {
  return useSessionContext().workspaces
}

// slug <-> id resolvers plus the full workspace-by-id lookup.
export function useWorkspaceResolver() {
  const { idBySlug, slugById, workspaceById } = useSessionContext()
  return { idBySlug, slugById, workspaceById }
}

export { useSetCurrentWorkspace } from './use-set-current-workspace'
