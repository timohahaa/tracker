import { useQueryClient } from '@tanstack/react-query'

import { $api } from '@/api/client'

import { CURRENT_USER_QUERY_KEY } from './use-current-user'
import { WORKSPACES_QUERY_KEY } from './use-workspaces'

// Opens a server-side session (POST /api/auth/login sets the httpOnly cookie).
// On success it invalidates the bootstrap queries and awaits their refetch so
// SessionProvider recomputes `authenticated` before the caller navigates.
export function useLogin() {
  const queryClient = useQueryClient()

  return $api.useMutation('post', '/api/auth/login', {
    onSuccess: async () => {
      await Promise.all([
        queryClient.invalidateQueries({ queryKey: CURRENT_USER_QUERY_KEY }),
        queryClient.invalidateQueries({ queryKey: WORKSPACES_QUERY_KEY }),
      ])
    },
  })
}
