export type SessionStatus = 'loading' | 'authenticated' | 'unauthenticated'

// TODO(phase-1.3): Replace this placeholder with the real auth-bootstrap
// provider. Phase 1.3 will resolve the session via `GET /api/auth/me`
// (see docs/tech/auth.md): 401 -> 'unauthenticated', otherwise 'authenticated'
// once the current user + workspace list are loaded. Until then this hook has
// no server dependency and must not trigger any request.
//
// Default is 'unauthenticated' so ProtectedRoute redirects to /login (part of
// the Phase 1.2 DoD). Flip this to 'authenticated' locally to smoke-test that
// protected routes render.
export const PLACEHOLDER_SESSION_STATUS: SessionStatus = 'unauthenticated'

export function useSessionStatus(): { status: SessionStatus } {
  return { status: PLACEHOLDER_SESSION_STATUS }
}
