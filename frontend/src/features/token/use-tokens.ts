import { useQueryClient } from '@tanstack/react-query'

import { $api } from '@/api/client'
import type { components } from '@/api/generated/schema'

export type ApiToken =
  components['schemas']['internal_handler_token.tokenResponse']
export type CreateTokenResponse =
  components['schemas']['internal_handler_token.createTokenResponse']
export type CreateTokenBody =
  components['schemas']['internal_handler_token.createTokenRequest']

// Shared query-key prefix (read + invalidation). openapi-react-query keys are
// `['get', path, params?]`, so this prefix matches every param variant.
export const TOKENS_QUERY_KEY = [
  'get',
  '/api/workspaces/{workspaceID}/tokens',
] as const

// GET the current user's API tokens. Access: authenticated. The raw token is
// NOT included — only { id, name, last_used_at?, created_at }.
export function useTokens(workspaceID: string) {
  return $api.useQuery('get', '/api/workspaces/{workspaceID}/tokens', {
    params: { path: { workspaceID } },
  })
}

// POST create an API token. The response includes the RAW token exactly once —
// callers must reveal it immediately and never persist it. Access: authenticated.
export function useCreateToken() {
  const queryClient = useQueryClient()

  return $api.useMutation('post', '/api/workspaces/{workspaceID}/tokens', {
    onSuccess: () =>
      queryClient.invalidateQueries({ queryKey: TOKENS_QUERY_KEY }),
  })
}

// DELETE (revoke) an API token. Access: authenticated (own tokens).
export function useDeleteToken() {
  const queryClient = useQueryClient()

  return $api.useMutation(
    'delete',
    '/api/workspaces/{workspaceID}/tokens/{tokenID}',
    {
      onSuccess: () =>
        queryClient.invalidateQueries({ queryKey: TOKENS_QUERY_KEY }),
    },
  )
}
