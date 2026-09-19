import { useQueryClient } from '@tanstack/react-query'

import { $api } from '@/api/client'

import { CURRENT_USER_QUERY_KEY } from './use-current-user'
import { WORKSPACES_QUERY_KEY } from './use-workspaces'

// Ends the server-side session (POST /api/auth/logout clears the cookie). On
// success it drops the cached user/workspaces so no stale session state lingers;
// the caller navigates to /login afterwards.
export function useLogout() {
  const queryClient = useQueryClient()

  return $api.useMutation('post', '/api/auth/logout', {
    onSuccess: () => {
      queryClient.removeQueries({ queryKey: CURRENT_USER_QUERY_KEY })
      queryClient.removeQueries({ queryKey: WORKSPACES_QUERY_KEY })
    },
  })
}
