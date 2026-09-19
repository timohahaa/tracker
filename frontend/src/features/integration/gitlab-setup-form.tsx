import { zodResolver } from '@hookform/resolvers/zod'
import { useForm } from 'react-hook-form'
import { toast } from 'sonner'
import { z } from 'zod'

import { AppError } from '@/api/client'
import { FieldError } from '@/components/common/field-error'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { useCurrentWorkspace } from '@/features/auth/workspace-context'
import { AuthErrorBanner } from '@/pages/auth-shell'

import { useSetupGitlab } from './use-gitlab'

// Client-side shape validation only. The backend validates the URL/token against
// GitLab and is the source of truth; server `fields[]` surface via <FieldError>.
const setupSchema = z.object({
  base_url: z.string().trim().url('Enter a valid URL (e.g. https://gitlab.com)'),
  access_token: z.string().trim().min(1, 'Access token is required'),
  webhook_secret: z.string().trim().min(1, 'Webhook secret is required'),
})

type SetupValues = z.infer<typeof setupSchema>

interface GitlabSetupFormProps {
  // Called after a successful configure/reconnect.
  onSuccess: () => void
  // Optional cancel affordance (shown when reconnecting an existing integration).
  onCancel?: () => void
  submitLabel: string
}

// GitLab connection form (base_url + access_token + webhook_secret). The stored
// credentials are never rendered — this form only ever writes new values.
export function GitlabSetupForm({
  onSuccess,
  onCancel,
  submitLabel,
}: GitlabSetupFormProps) {
  const { id: workspaceID } = useCurrentWorkspace()
  const setup = useSetupGitlab()

  const {
    register,
    handleSubmit,
    formState: { errors },
  } = useForm<SetupValues>({
    resolver: zodResolver(setupSchema),
    defaultValues: { base_url: '', access_token: '', webhook_secret: '' },
  })

  const serverError = setup.error instanceof AppError ? setup.error : null
  // 403 (member) and other non-field errors surface as a banner; 400 validation
  // with fields[] renders under the inputs.
  const bannerMessage =
    serverError &&
    (serverError.code !== 'validation' || serverError.fields.length === 0)
      ? serverError.message
      : null

  const onSubmit = (values: SetupValues) => {
    setup.mutate(
      { params: { path: { workspaceID } }, body: values },
      {
        onSuccess: () => {
          toast.success('GitLab integration saved')
          onSuccess()
        },
      },
    )
  }

  return (
    <form onSubmit={handleSubmit(onSubmit)} className="space-y-4" noValidate>
      {bannerMessage ? <AuthErrorBanner message={bannerMessage} /> : null}

      <div className="space-y-1.5">
        <label htmlFor="gitlab-base-url" className="text-sm font-medium">
          GitLab URL
        </label>
        <Input
          id="gitlab-base-url"
          placeholder="https://gitlab.com"
          autoComplete="off"
          aria-invalid={!!errors.base_url}
          {...register('base_url')}
        />
        {errors.base_url ? (
          <p className="text-sm text-destructive">{errors.base_url.message}</p>
        ) : (
          <FieldError error={setup.error} name="base_url" />
        )}
      </div>

      <div className="space-y-1.5">
        <label htmlFor="gitlab-access-token" className="text-sm font-medium">
          Access token
        </label>
        <Input
          id="gitlab-access-token"
          type="password"
          autoComplete="off"
          aria-invalid={!!errors.access_token}
          {...register('access_token')}
        />
        <p className="text-xs text-muted-foreground">
          A GitLab personal or project access token. Stored encrypted and never
          shown again.
        </p>
        {errors.access_token ? (
          <p className="text-sm text-destructive">
            {errors.access_token.message}
          </p>
        ) : (
          <FieldError error={setup.error} name="access_token" />
        )}
      </div>

      <div className="space-y-1.5">
        <label htmlFor="gitlab-webhook-secret" className="text-sm font-medium">
          Webhook secret
        </label>
        <Input
          id="gitlab-webhook-secret"
          type="password"
          autoComplete="off"
          aria-invalid={!!errors.webhook_secret}
          {...register('webhook_secret')}
        />
        <p className="text-xs text-muted-foreground">
          Used to verify incoming GitLab webhook events.
        </p>
        {errors.webhook_secret ? (
          <p className="text-sm text-destructive">
            {errors.webhook_secret.message}
          </p>
        ) : (
          <FieldError error={setup.error} name="webhook_secret" />
        )}
      </div>

      <div className="flex gap-2">
        <Button type="submit" disabled={setup.isPending}>
          {setup.isPending ? 'Saving…' : submitLabel}
        </Button>
        {onCancel ? (
          <Button
            type="button"
            variant="outline"
            onClick={onCancel}
            disabled={setup.isPending}
          >
            Cancel
          </Button>
        ) : null}
      </div>
    </form>
  )
}
