import { useQueryClient } from '@tanstack/react-query'

import { $api } from '@/api/client'

import { CURRENT_USER_QUERY_KEY } from './use-current-user'

// Persists the user's active workspace server-side (PATCH /api/auth/me), so the
// "resume where you left off" default follows them across devices. Invalidates
// the current-user query so `settings.current_workspace_id` is refetched.
export function useSetCurrentWorkspace() {
  const queryClient = useQueryClient()

  return $api.useMutation('patch', '/api/auth/me', {
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: CURRENT_USER_QUERY_KEY })
    },
  })
}
