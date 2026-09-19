import { useQueryClient } from '@tanstack/react-query'

import { $api } from '@/api/client'
import type { components } from '@/api/generated/schema'
import { WORKSPACES_QUERY_KEY } from '@/features/auth/use-workspaces'

export type WorkspaceDetail =
  components['schemas']['internal_handler_workspace.workspaceResponse']
export type UpdateWorkspaceBody =
  components['schemas']['internal_handler_workspace.updateWorkspaceRequest']

// Shared query-key prefix for the single-workspace query. openapi-react-query
// keys are `['get', path, params?]`, so this prefix matches every param variant
// for invalidation.
export const WORKSPACE_QUERY_KEY = [
  'get',
  '/api/workspaces/{workspaceID}',
] as const

// GET one workspace (name/slug/timestamps). Access: any member.
export function useWorkspace(workspaceID: string) {
  return $api.useQuery('get', '/api/workspaces/{workspaceID}', {
    params: { path: { workspaceID } },
  })
}

// PATCH workspace settings (name/slug). Access: owner/admin. Invalidates both
// the single-workspace query and the workspaces list (its slug<->id map feeds
// routing) so a slug change is reflected everywhere. Callers handle navigation
// to the new slug in their own onSuccess.
export function useUpdateWorkspace() {
  const queryClient = useQueryClient()

  return $api.useMutation('patch', '/api/workspaces/{workspaceID}', {
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: WORKSPACE_QUERY_KEY })
      queryClient.invalidateQueries({ queryKey: WORKSPACES_QUERY_KEY })
    },
  })
}
