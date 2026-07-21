// Client-side domain error. Mirrors the backend error envelope
// `{ error: { code, message, detail, fields[] } }` (see docs/tech/error-handling.md),
// unpacked into a flat shape used across the whole frontend.

export type AppErrorCode =
  | 'not_found'
  | 'forbidden'
  | 'unauthorized'
  | 'validation'
  | 'already_exists'
  | 'internal'

export interface AppErrorField {
  field: string
  reason: string
  message: string
}

export class AppError extends Error {
  readonly code: AppErrorCode
  readonly detail?: string
  readonly fields: AppErrorField[]
  readonly status?: number

  constructor(params: {
    code: AppErrorCode
    message: string
    detail?: string
    fields?: AppErrorField[]
    status?: number
  }) {
    super(params.message)
    this.name = 'AppError'
    this.code = params.code
    this.detail = params.detail
    this.fields = params.fields ?? []
    this.status = params.status
  }
}

// Server error body shape (the value under the top-level `error` key).
interface ServerErrorBody {
  code?: string
  message?: string
  detail?: string
  fields?: Array<{ field?: string; reason?: string; message?: string }>
}

interface ServerErrorEnvelope {
  error?: ServerErrorBody
}

const KNOWN_CODES: readonly AppErrorCode[] = [
  'not_found',
  'forbidden',
  'unauthorized',
  'validation',
  'already_exists',
  'internal',
]

function toAppErrorCode(code: string | undefined): AppErrorCode {
  return KNOWN_CODES.includes(code as AppErrorCode)
    ? (code as AppErrorCode)
    : 'internal'
}

function isEnvelope(data: unknown): data is ServerErrorEnvelope {
  return (
    typeof data === 'object' &&
    data !== null &&
    'error' in data &&
    typeof (data as ServerErrorEnvelope).error === 'object'
  )
}

// Normalizes an unknown failure (typically an AxiosError) into an AppError.
// Falls back to `internal` when the payload does not match the server envelope.
export function normalizeError(
  responseData: unknown,
  status?: number,
  fallbackMessage = 'Something went wrong',
): AppError {
  if (isEnvelope(responseData) && responseData.error) {
    const body = responseData.error
    return new AppError({
      code: toAppErrorCode(body.code),
      message: body.message ?? fallbackMessage,
      detail: body.detail,
      fields: (body.fields ?? []).map((f) => ({
        field: f.field ?? '',
        reason: f.reason ?? '',
        message: f.message ?? '',
      })),
      status,
    })
  }

  return new AppError({
    code: 'internal',
    message: fallbackMessage,
    status,
  })
}
