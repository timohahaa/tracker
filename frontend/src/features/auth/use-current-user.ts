import { $api } from '@/api/client'
import type { components } from '@/api/generated/schema'

export type CurrentUser = components['schemas']['internal_handler_auth.userResponse']

// Query key for the current-user bootstrap request. Kept as a shared constant
// so the session provider, mutations and the global 401 handler can target it.
export const CURRENT_USER_QUERY_KEY = ['get', '/api/auth/me'] as const

// Bootstrap query for the logged-in user (session-cookie resume). `retry: false`
// so a 401 (no/expired session) resolves immediately instead of retrying —
// this is the signal the session provider reads to mark 'unauthenticated'.
export function useCurrentUserQuery() {
  return $api.useQuery('get', '/api/auth/me', {}, { retry: false })
}
