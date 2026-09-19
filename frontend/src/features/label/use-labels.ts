import { $api } from '@/api/client'
import type { components } from '@/api/generated/schema'

export type Label =
  components['schemas']['internal_handler_label.labelResponse']

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
