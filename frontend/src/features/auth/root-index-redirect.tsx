import { Navigate } from 'react-router-dom'

import { ErrorState } from '@/components/common/error-state'
import { RouteFallback } from '@/components/common/skeletons'

import { useSession } from './session-provider'

// Landing redirect for `/`. Resolves the session and sends the user to their
// default workspace slug when authenticated, or to /login otherwise.
export function RootIndexRedirect() {
  const { status, error, defaultWorkspace } = useSession()

  if (error) {
    return (
      <div className="flex min-h-svh w-full items-center justify-center p-6">
        <div className="w-full max-w-md">
          <ErrorState error={error} />
        </div>
      </div>
    )
  }

  if (status === 'loading') return <RouteFallback />

  if (status === 'authenticated' && defaultWorkspace?.slug) {
    return <Navigate to={`/${defaultWorkspace.slug}`} replace />
  }

  return <Navigate to="/login" replace />
}
