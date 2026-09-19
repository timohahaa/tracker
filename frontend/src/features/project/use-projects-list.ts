import { $api } from '@/api/client'
import type { components } from '@/api/generated/schema'

export type Project =
  components['schemas']['internal_handler_project.projectResponse']

// Shared query-key prefix (read + invalidation).
export const PROJECTS_QUERY_KEY = [
  'get',
  '/api/workspaces/{workspaceID}/projects',
] as const

// Flat, single-page project list for pickers (issue project filter here; issue
// create/edit in 2.4.2). 2.5 adds the infinite-scroll `useProjects` for the
// projects page proper — this thin hook stays for selection contexts.
export function useProjectsList(workspaceID: string) {
  return $api.useQuery('get', '/api/workspaces/{workspaceID}/projects', {
    params: { path: { workspaceID }, query: { limit: 100 } },
  })
}
