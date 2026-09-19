import { useQueryClient } from '@tanstack/react-query'

import { $api } from '@/api/client'

import { CURRENT_USER_QUERY_KEY } from './use-current-user'
import { WORKSPACES_QUERY_KEY } from './use-workspaces'

// Creates a user + workspace and opens a session (POST /api/auth/register).
// Mirrors useLogin: invalidate and await the bootstrap queries so the new
// workspace is resolvable by slug before the caller navigates into it.
export function useRegister() {
  const queryClient = useQueryClient()

  return $api.useMutation('post', '/api/auth/register', {
    onSuccess: async () => {
      await Promise.all([
        queryClient.invalidateQueries({ queryKey: CURRENT_USER_QUERY_KEY }),
        queryClient.invalidateQueries({ queryKey: WORKSPACES_QUERY_KEY }),
      ])
    },
  })
}
