import { QueryClientProvider } from '@tanstack/react-query'
import { RouterProvider } from 'react-router-dom'

import { queryClient } from './query-client'
import { router } from './router'

// Cross-cutting providers wired in Phase 0: server-state (TanStack Query) and
// routing (React Router). ThemeProvider / auth-bootstrap arrive in Phase 1.
export function Providers() {
  return (
    <QueryClientProvider client={queryClient}>
      <RouterProvider router={router} />
    </QueryClientProvider>
  )
}
