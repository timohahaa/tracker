import { QueryClientProvider } from '@tanstack/react-query'
import { RouterProvider } from 'react-router-dom'

import { ThemeProvider } from '@/components/theme-provider'
import { Toaster } from '@/components/ui/sonner'

import { queryClient } from './query-client'
import { router } from './router'

// Cross-cutting providers. Phase 0 wired server-state (TanStack Query) and
// routing (React Router); Phase 1.1 adds the ThemeProvider (dark by default)
// and the sonner Toaster. Auth-bootstrap arrives in Phase 1.3.
export function Providers() {
  return (
    <ThemeProvider>
      <QueryClientProvider client={queryClient}>
        <RouterProvider router={router} />
        <Toaster />
      </QueryClientProvider>
    </ThemeProvider>
  )
}
