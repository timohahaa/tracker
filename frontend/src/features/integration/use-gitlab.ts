import { useQueryClient } from '@tanstack/react-query'

import { $api } from '@/api/client'
import type { components } from '@/api/generated/schema'

export type GitlabIntegration =
  components['schemas']['internal_handler_integration.gitlabResponse']
export type SetupGitlabBody =
  components['schemas']['internal_handler_integration.setupGitLabRequest']

// Shared query-key prefix (read + invalidation). openapi-react-query keys are
// `['get', path, params?]`, so this prefix matches every param variant.
export const GITLAB_QUERY_KEY = [
  'get',
  '/api/workspaces/{workspaceID}/integrations/gitlab',
] as const

// GET GitLab integration status. Access: any member. Credentials are never
// returned by the backend — only `connected` + `settings.base_url` + timestamps.
export function useGitlabIntegration(workspaceID: string) {
  return $api.useQuery(
    'get',
    '/api/workspaces/{workspaceID}/integrations/gitlab',
    { params: { path: { workspaceID } } },
  )
}

// POST configure/reconnect the GitLab integration (base_url + access_token +
// webhook_secret). Access: owner/admin. Errors: 400 validation, 403.
export function useSetupGitlab() {
  const queryClient = useQueryClient()

  return $api.useMutation(
    'post',
    '/api/workspaces/{workspaceID}/integrations/gitlab',
    {
      onSuccess: () =>
        queryClient.invalidateQueries({ queryKey: GITLAB_QUERY_KEY }),
    },
  )
}

// DELETE the GitLab integration. Access: owner/admin.
export function useDeleteGitlab() {
  const queryClient = useQueryClient()

  return $api.useMutation(
    'delete',
    '/api/workspaces/{workspaceID}/integrations/gitlab',
    {
      onSuccess: () =>
        queryClient.invalidateQueries({ queryKey: GITLAB_QUERY_KEY }),
    },
  )
}
