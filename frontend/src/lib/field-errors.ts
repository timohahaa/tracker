import { AppError } from '@/lib/errors'

// Pulls the message for a single field out of an AppError's `fields[]`.
// Form-agnostic on purpose: Phase 2.1 forms (react-hook-form + zod) map
// server-side `fields[]` onto their inputs through this helper.
export function getFieldError(
  error: unknown,
  name: string,
): string | undefined {
  if (!(error instanceof AppError)) return undefined
  return error.fields.find((f) => f.field === name)?.message || undefined
}
