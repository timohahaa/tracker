import { toast } from 'sonner'

import { AppError } from '@/api/client'

// Thin wrapper over sonner's error toast that pulls a human message out of an
// AppError (the normalized shape every mutation rejects with), falling back to a
// generic message for anything else. Keeps failure feedback consistent without
// introducing a second toast system.
export function toastError(err: unknown, fallback = 'Something went wrong') {
  toast.error(err instanceof AppError ? err.message : fallback)
}
