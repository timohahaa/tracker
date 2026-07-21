import { QueryClient } from '@tanstack/react-query'

// Sane defaults for a session-cookie SPA. Errors are already normalized to
// AppError by the openapi-fetch middleware (see src/api/client.ts).
export const queryClient = new QueryClient({
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
