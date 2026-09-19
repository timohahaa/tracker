import { zodResolver } from '@hookform/resolvers/zod'
import { AlertTriangle, Check, Copy } from 'lucide-react'
import { type ReactNode, useState } from 'react'
import { useForm } from 'react-hook-form'
import { toast } from 'sonner'
import { z } from 'zod'

import { AppError } from '@/api/client'
import { FieldError } from '@/components/common/field-error'
import { Button } from '@/components/ui/button'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from '@/components/ui/dialog'
import { Input } from '@/components/ui/input'
import { useCurrentWorkspace } from '@/features/auth/workspace-context'
import { AuthErrorBanner } from '@/pages/auth-shell'

import { useCreateToken } from './use-tokens'

const tokenSchema = z.object({
  name: z.string().trim().min(1, 'Name is required'),
})

type TokenValues = z.infer<typeof tokenSchema>

// Create an API token, then reveal the raw token EXACTLY ONCE in a copy panel.
// The plaintext lives only in local component state and is cleared when the
// dialog closes — it is never persisted or re-fetchable afterwards.
export function CreateTokenDialog({ trigger }: { trigger: ReactNode }) {
  const { id: workspaceID } = useCurrentWorkspace()
  const createToken = useCreateToken()
  const [open, setOpen] = useState(false)
  // The one-time plaintext token (set on success, cleared on close).
  const [createdToken, setCreatedToken] = useState<string | null>(null)
  const [copied, setCopied] = useState(false)

  const {
    register,
    handleSubmit,
    reset,
    formState: { errors },
  } = useForm<TokenValues>({
    resolver: zodResolver(tokenSchema),
    defaultValues: { name: '' },
  })

  const handleOpenChange = (next: boolean) => {
    setOpen(next)
    if (!next) {
      // Drop the plaintext token from memory as soon as the dialog closes.
      setCreatedToken(null)
      setCopied(false)
      reset({ name: '' })
      createToken.reset()
    }
  }

  const serverError =
    createToken.error instanceof AppError ? createToken.error : null
  const bannerMessage =
    serverError &&
    (serverError.code !== 'validation' || serverError.fields.length === 0)
      ? serverError.message
      : null

  const onSubmit = (values: TokenValues) => {
    createToken.mutate(
      { params: { path: { workspaceID } }, body: { name: values.name.trim() } },
      {
        onSuccess: (data) => {
          if (data.token) {
            setCreatedToken(data.token)
          } else {
            // Defensive: if the backend ever omits the raw token, close out
            // rather than leaving a broken reveal panel.
            toast.success('Token created')
            handleOpenChange(false)
          }
        },
      },
    )
  }

  const handleCopy = async () => {
    if (!createdToken) return
    try {
      await navigator.clipboard.writeText(createdToken)
      setCopied(true)
      toast.success('Token copied to clipboard')
    } catch {
      toast.error('Could not copy — select and copy the token manually')
    }
  }

  return (
    <Dialog open={open} onOpenChange={handleOpenChange}>
      <DialogTrigger asChild>{trigger}</DialogTrigger>
      <DialogContent>
        {createdToken ? (
          <>
            <DialogHeader>
              <DialogTitle>Copy your API token</DialogTitle>
              <DialogDescription>
                This is the only time the token will be shown. Copy it now and
                store it somewhere safe.
              </DialogDescription>
            </DialogHeader>

            <div
              role="alert"
              className="flex items-start gap-2 rounded-md border border-destructive/50 bg-destructive/10 p-3 text-sm text-destructive"
            >
              <AlertTriangle
                className="mt-0.5 h-4 w-4 shrink-0"
                aria-hidden="true"
              />
              <p>You won&apos;t be able to see this token again after closing.</p>
            </div>

            <div className="flex items-center gap-2">
              <code className="min-w-0 flex-1 truncate rounded-md border bg-muted px-3 py-2 font-mono text-sm">
                {createdToken}
              </code>
              <Button
                type="button"
                variant="outline"
                size="icon"
                onClick={handleCopy}
                aria-label="Copy token"
              >
                {copied ? (
                  <Check className="h-4 w-4 text-emerald-500" />
                ) : (
                  <Copy className="h-4 w-4" />
                )}
              </Button>
            </div>

            <DialogFooter>
              <Button type="button" onClick={() => handleOpenChange(false)}>
                Done
              </Button>
            </DialogFooter>
          </>
        ) : (
          <>
            <DialogHeader>
              <DialogTitle>Create API token</DialogTitle>
              <DialogDescription>
                Generate a personal token for the API and MCP. Use it as a Bearer
                token in the Authorization header.
              </DialogDescription>
            </DialogHeader>

            <form
              onSubmit={handleSubmit(onSubmit)}
              className="space-y-4"
              noValidate
            >
              {bannerMessage ? <AuthErrorBanner message={bannerMessage} /> : null}

              <div className="space-y-1.5">
                <label htmlFor="token-name" className="text-sm font-medium">
                  Name
                </label>
                <Input
                  id="token-name"
                  placeholder="e.g. CI pipeline"
                  aria-invalid={!!errors.name}
                  {...register('name')}
                />
                {errors.name ? (
                  <p className="text-sm text-destructive">
                    {errors.name.message}
                  </p>
                ) : (
                  <FieldError error={createToken.error} name="name" />
                )}
              </div>

              <DialogFooter>
                <Button
                  type="button"
                  variant="outline"
                  onClick={() => handleOpenChange(false)}
                  disabled={createToken.isPending}
                >
                  Cancel
                </Button>
                <Button type="submit" disabled={createToken.isPending}>
                  {createToken.isPending ? 'Creating…' : 'Create token'}
                </Button>
              </DialogFooter>
            </form>
          </>
        )}
      </DialogContent>
    </Dialog>
  )
}
