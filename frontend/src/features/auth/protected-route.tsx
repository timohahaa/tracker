import { Navigate, Outlet } from 'react-router-dom'

import { RouteFallback } from '@/components/common/skeletons'
import { useSessionStatus } from '@/features/auth/use-session-status'

// Guard for authenticated route subtrees. Rendered as a layout route, so it
// yields an <Outlet /> for its children when the session is valid.
//
// TODO(phase-1.3): the redirect target may need to preserve the attempted URL
// (e.g. `?next=`) once real session resume lands.
export function ProtectedRoute() {
  const { status } = useSessionStatus()

  if (status === 'loading') return <RouteFallback />
  if (status === 'unauthenticated') return <Navigate to="/login" replace />

  return <Outlet />
}
