import {
  isRouteErrorResponse,
  Link,
  useRouteError,
} from 'react-router-dom'

import { AppError } from '@/api/client'
import { ErrorState } from '@/components/common/error-state'
import { buttonVariants } from '@/components/ui/button'
import { cn } from '@/lib/utils'

// Router-level errorElement. Catches render/loader errors anywhere in the
// route subtree and renders them through the shared ErrorState primitive.
export function RouteErrorBoundary() {
  const error = useRouteError()

  let content

  if (error instanceof AppError) {
    // Unauthorized gets an explicit path back to login.
    const isUnauthorized = error.code === 'unauthorized'
    content = (
      <ErrorState
        error={error}
        onRetry={isUnauthorized ? undefined : () => window.location.reload()}
      />
    )
    if (isUnauthorized) {
      content = (
        <div className="flex flex-col items-center gap-4">
          <ErrorState error={error} />
          <Link
            to="/login"
            className={cn(buttonVariants({ variant: 'default' }))}
          >
            Go to login
          </Link>
        </div>
      )
    }
  } else if (isRouteErrorResponse(error)) {
    // Router-thrown responses (e.g. a route that throws a Response / 404).
    const message =
      typeof error.data === 'string' && error.data
        ? error.data
        : 'The page could not be loaded.'
    content = (
      <ErrorState
        error={new Error(message)}
        title={`${error.status} ${error.statusText}`.trim()}
      />
    )
  } else {
    content = <ErrorState error={error} />
  }

  return (
    <div className="flex min-h-svh w-full items-center justify-center p-6">
      <div className="w-full max-w-md">{content}</div>
    </div>
  )
}
