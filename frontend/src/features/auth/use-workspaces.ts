import { $api } from '@/api/client'
import type { components } from '@/api/generated/schema'

export type Workspace =
  components['schemas']['internal_handler_workspace.workspaceMineResponse']

// Query key for the workspaces list. Shared so the global 401 handler can drop
// it from the cache on logout.
export const WORKSPACES_QUERY_KEY = ['get', '/api/workspaces'] as const

interface UseWorkspacesOptions {
  // Gated on an authenticated `me` so we never hit this endpoint before the
  // session is confirmed.
  enabled: boolean
}

export function useWorkspacesQuery({ enabled }: UseWorkspacesOptions) {
  return $api.useQuery('get', '/api/workspaces', {}, { enabled })
}
