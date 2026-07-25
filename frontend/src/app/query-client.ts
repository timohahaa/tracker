import { QueryCache, QueryClient } from '@tanstack/react-query'

import { AppError } from '@/api/client'
import { CURRENT_USER_QUERY_KEY } from '@/features/auth/use-current-user'
import { WORKSPACES_QUERY_KEY } from '@/features/auth/use-workspaces'

import { router } from './router'

// Runtime 401 handling: if a protected query fails because the session expired
// mid-session, drop the cached user/workspaces and send the user to /login
// (preserving the attempted URL). The bootstrap `me` query is intentionally
// skipped here — it is owned by SessionProvider, which drives the initial
// authenticated/unauthenticated decision (and must not kick anyone off /login
// or /register).
function handleGlobalQueryError(error: unknown, queryKey: readonly unknown[]) {
  if (queryKey[1] === '/api/auth/me') return
  if (!(error instanceof AppError) || error.code !== 'unauthorized') return

  queryClient.removeQueries({ queryKey: CURRENT_USER_QUERY_KEY })
  queryClient.removeQueries({ queryKey: WORKSPACES_QUERY_KEY })

  if (window.location.pathname === '/login') return
  const next = encodeURIComponent(
    window.location.pathname + window.location.search,
  )
  void router.navigate(`/login?next=${next}`, { replace: true })
}

// Sane defaults for a session-cookie SPA. Errors are already normalized to
// AppError by the openapi-fetch middleware (see src/api/client.ts).
export const queryClient = new QueryClient({
  queryCache: new QueryCache({
    onError: (error, query) => handleGlobalQueryError(error, query.queryKey),
  }),
  defaultOptions: {
    queries: {
      staleTime: 30_000,
      retry: 1,
      refetchOnWindowFocus: false,
    },
    mutations: {
      retry: 0,
    },
  },
})
