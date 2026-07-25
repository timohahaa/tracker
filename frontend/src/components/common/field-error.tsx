import { getFieldError } from '@/lib/field-errors'
import { cn } from '@/lib/utils'

interface FieldErrorProps {
  /** Any thrown value; only AppError with matching `fields[]` renders. */
  error: unknown
  /** Field name to match against `AppErrorField.field`. */
  name: string
  className?: string
}

// Inline, per-input error text (rendered under a form control). Renders
// nothing when there is no matching field error.
export function FieldError({ error, name, className }: FieldErrorProps) {
  const message = getFieldError(error, name)
  if (!message) return null

  return (
    <p className={cn('text-sm text-destructive', className)}>{message}</p>
  )
}
