import { AlertTriangle } from 'lucide-react'

import { AppError } from '@/api/client'
import { Button } from '@/components/ui/button'
import { cn } from '@/lib/utils'

interface ErrorStateProps {
  /** An AppError renders its code/message/detail/fields; anything else falls
   *  back to a generic message. */
  error: unknown
  /** Overrides the derived heading. */
  title?: string
  /** Shows a retry button when provided. */
  onRetry?: () => void
  className?: string
}

const TITLE_BY_CODE: Record<AppError['code'], string> = {
  not_found: 'Not found',
  forbidden: 'Access denied',
  unauthorized: 'Sign in required',
  validation: 'Invalid request',
  already_exists: 'Already exists',
  internal: 'Something went wrong',
}

// Generic, full-block error renderer for the AppError shape. Used by route
// error boundaries and by data views (query errors) across the app.
export function ErrorState({
  error,
  title,
  onRetry,
  className,
}: ErrorStateProps) {
  const appError = error instanceof AppError ? error : null
  const heading =
    title ?? (appError ? TITLE_BY_CODE[appError.code] : 'Something went wrong')
  const message =
    appError?.message ??
    (error instanceof Error ? error.message : 'An unexpected error occurred.')

  return (
    <div
      className={cn(
        'flex flex-col items-center justify-center gap-3 rounded-lg border border-dashed p-8 text-center',
        className,
      )}
      role="alert"
    >
      <AlertTriangle
        className="h-8 w-8 text-destructive"
        aria-hidden="true"
      />
      <div className="space-y-1">
        <h2 className="text-lg font-semibold">{heading}</h2>
        <p className="text-sm text-muted-foreground">{message}</p>
        {appError?.detail ? (
          <p className="text-xs text-muted-foreground">{appError.detail}</p>
        ) : null}
      </div>

      {appError && appError.fields.length > 0 ? (
        <ul className="w-full max-w-sm space-y-1 text-left text-sm text-destructive">
          {appError.fields.map((f, i) => (
            <li key={`${f.field}-${i}`}>
              <span className="font-medium">{f.field}</span>: {f.message}
            </li>
          ))}
        </ul>
      ) : null}

      {onRetry ? (
        <Button variant="outline" size="sm" onClick={onRetry}>
          Try again
        </Button>
      ) : null}
    </div>
  )
}
