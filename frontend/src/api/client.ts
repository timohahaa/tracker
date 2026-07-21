import createFetchClient, { type Middleware } from 'openapi-fetch'
import createQueryClient from 'openapi-react-query'

import type { paths } from '@/api/generated/schema'
import { AppError, normalizeError } from '@/lib/errors'

// Fetch-based API client (replaces the previous axios instance + orval mutator).
//
// - `credentials: 'include'`: web auth is a server-side session in an httpOnly
//   cookie. The client never reads/stores the token; the cookie rides along
//   automatically (in dev via the Vite same-origin proxy, see vite.config.ts).
// - No `baseUrl`: OpenAPI paths already include the `/api` prefix, so requests
//   stay relative to the current origin and hit the dev proxy.
export const fetchClient = createFetchClient<paths>({
  credentials: 'include',
})

// Normalize every failure into an AppError so the rest of the frontend deals
// with a single error shape.
const errorMiddleware: Middleware = {
  // Non-2xx: throwing here propagates straight out of openapi-fetch (onResponse
  // runs outside its internal try/catch), so React Query receives an AppError.
  async onResponse({ response }) {
    if (response.ok) return
    let body: unknown
    try {
      body = await response.clone().json()
    } catch {
      body = undefined
    }
    throw normalizeError(body, response.status, response.statusText)
  },
  // Network / aborted / unexpected failures: openapi-fetch expects an Error to
  // be returned here, which it then throws.
  onError({ error }) {
    if (error instanceof AppError) return error
    const message = error instanceof Error ? error.message : undefined
    return normalizeError(undefined, undefined, message)
  },
}

fetchClient.use(errorMiddleware)

// Typed React Query hooks built from the OpenAPI `paths` types. Request
// cancellation is handled by React Query's AbortSignal — no manual wiring.
export const $api = createQueryClient(fetchClient)

// Single source of truth for the domain error shape across the frontend.
export { AppError, normalizeError }
export type { AppErrorCode, AppErrorField } from '@/lib/errors'
