import { useQueryClient } from '@tanstack/react-query'

import { $api } from '@/api/client'
import type { components } from '@/api/generated/schema'
import { ISSUE_QUERY_KEY } from '@/features/issue/use-issue'
import { ISSUES_QUERY_KEY } from '@/features/issue/use-issues'

export type Label =
  components['schemas']['internal_handler_label.labelResponse']
export type CreateLabelBody =
  components['schemas']['internal_handler_label.createLabelRequest']
export type UpdateLabelBody =
  components['schemas']['internal_handler_label.updateLabelRequest']

// Shared query-key prefix (read + invalidation). 2.6 (label CRUD) extends this.
export const LABELS_QUERY_KEY = [
  'get',
  '/api/workspaces/{workspaceID}/labels',
] as const

// GET all labels (workspace + team level). Access: any member. Used here for the
// issue-list label filter; 2.4.3/2.6 reuse it.
export function useLabels(workspaceID: string) {
  return $api.useQuery('get', '/api/workspaces/{workspaceID}/labels', {
    params: { path: { workspaceID } },
  })
}

// POST create a label. `team_id` omitted → workspace-level; set → team-scoped.
// Access: any member. Errors: 400 (color/name), 409 (duplicate name in scope).
export function useCreateLabel() {
  const queryClient = useQueryClient()

  return $api.useMutation('post', '/api/workspaces/{workspaceID}/labels', {
    onSuccess: () =>
      queryClient.invalidateQueries({ queryKey: LABELS_QUERY_KEY }),
  })
}

// PATCH a label (name/color). Access: any member. Invalidates the labels list so
// the issue label picker stays fresh, plus issue queries whose embedded label
// chips reflect the new name/color.
export function useUpdateLabel() {
  const queryClient = useQueryClient()

  return $api.useMutation(
    'patch',
    '/api/workspaces/{workspaceID}/labels/{labelID}',
    {
      onSuccess: () => {
        queryClient.invalidateQueries({ queryKey: LABELS_QUERY_KEY })
        queryClient.invalidateQueries({ queryKey: ISSUES_QUERY_KEY })
        queryClient.invalidateQueries({ queryKey: ISSUE_QUERY_KEY })
      },
    },
  )
}

// DELETE a label. Access: any member. The backend detaches it from issues, so
// invalidate issue queries alongside the labels list.
export function useDeleteLabel() {
  const queryClient = useQueryClient()

  return $api.useMutation(
    'delete',
    '/api/workspaces/{workspaceID}/labels/{labelID}',
    {
      onSuccess: () => {
        queryClient.invalidateQueries({ queryKey: LABELS_QUERY_KEY })
        queryClient.invalidateQueries({ queryKey: ISSUES_QUERY_KEY })
        queryClient.invalidateQueries({ queryKey: ISSUE_QUERY_KEY })
      },
    },
  )
}
