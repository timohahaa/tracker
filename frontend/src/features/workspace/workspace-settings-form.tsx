import { zodResolver } from '@hookform/resolvers/zod'
import { useQueryClient } from '@tanstack/react-query'
import { useForm } from 'react-hook-form'
import { useNavigate } from 'react-router-dom'
import { toast } from 'sonner'
import { z } from 'zod'

import { AppError } from '@/api/client'
import { FieldError } from '@/components/common/field-error'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { WORKSPACES_QUERY_KEY } from '@/features/auth/use-workspaces'
import { useCurrentWorkspace } from '@/features/auth/workspace-context'
import { AuthErrorBanner } from '@/pages/auth-shell'

import {
  useUpdateWorkspace,
  type WorkspaceDetail,
} from './use-workspace'

// Client-side shape validation only. The backend slugifies the slug (normalizes
// to url-safe form) and is the source of truth for the final value/format, so we
// keep this lenient and surface server `fields[]` via <FieldError>.
const settingsSchema = z.object({
  name: z.string().min(1, 'Name is required'),
  slug: z.string().min(1, 'Slug is required'),
})

type SettingsValues = z.infer<typeof settingsSchema>

interface WorkspaceSettingsFormProps {
  workspace: WorkspaceDetail
  // Owner/admin only. Members get a read-only view.
  canEdit: boolean
}

export function WorkspaceSettingsForm({
  workspace,
  canEdit,
}: WorkspaceSettingsFormProps) {
  const { id: workspaceID, slug: currentSlug } = useCurrentWorkspace()
  const navigate = useNavigate()
  const queryClient = useQueryClient()
  const updateWorkspace = useUpdateWorkspace()

  const serverError =
    updateWorkspace.error instanceof AppError ? updateWorkspace.error : null

  const {
    register,
    handleSubmit,
    reset,
    formState: { errors, isDirty },
  } = useForm<SettingsValues>({
    resolver: zodResolver(settingsSchema),
    values: {
      name: workspace.name ?? '',
      slug: workspace.slug ?? '',
    },
  })

  const onSubmit = (formValues: SettingsValues) => {
    updateWorkspace.mutate(
      { params: { path: { workspaceID } }, body: formValues },
      {
        onSuccess: async (data) => {
          toast.success('Workspace settings saved')
          reset({ name: data.name ?? '', slug: data.slug ?? '' })

          // A slug change re-routes: the URL slug is now stale. Wait for the
          // workspaces list (slug<->id resolver) to refetch before navigating,
          // otherwise ProtectedRoute can't resolve the new slug yet.
          const newSlug = data.slug
          if (newSlug && newSlug !== currentSlug) {
            await queryClient.invalidateQueries({
              queryKey: WORKSPACES_QUERY_KEY,
            })
            navigate(`/${newSlug}/settings`, { replace: true })
          }
        },
      },
    )
  }

  // 409 (slug taken) and other non-field errors surface as a banner; 400
  // validation with fields[] renders under the inputs.
  const bannerMessage =
    serverError &&
    (serverError.code !== 'validation' || serverError.fields.length === 0)
      ? serverError.message
      : null

  if (!canEdit) {
    return (
      <div className="space-y-4">
        <p className="text-sm text-muted-foreground">
          Only workspace owners and admins can change these settings.
        </p>
        <dl className="space-y-4">
          <div className="space-y-1">
            <dt className="text-sm font-medium">Name</dt>
            <dd className="text-sm text-muted-foreground">
              {workspace.name || '—'}
            </dd>
          </div>
          <div className="space-y-1">
            <dt className="text-sm font-medium">URL slug</dt>
            <dd className="font-mono text-sm text-muted-foreground">
              {workspace.slug || '—'}
            </dd>
          </div>
        </dl>
      </div>
    )
  }

  return (
    <form onSubmit={handleSubmit(onSubmit)} className="space-y-4" noValidate>
      {bannerMessage ? <AuthErrorBanner message={bannerMessage} /> : null}

      <div className="space-y-1.5">
        <label htmlFor="name" className="text-sm font-medium">
          Name
        </label>
        <Input id="name" aria-invalid={!!errors.name} {...register('name')} />
        {errors.name ? (
          <p className="text-sm text-destructive">{errors.name.message}</p>
        ) : (
          <FieldError error={updateWorkspace.error} name="name" />
        )}
      </div>

      <div className="space-y-1.5">
        <label htmlFor="slug" className="text-sm font-medium">
          URL slug
        </label>
        <Input
          id="slug"
          className="font-mono"
          aria-invalid={!!errors.slug}
          {...register('slug')}
        />
        <p className="text-xs text-muted-foreground">
          Used in this workspace&apos;s URL. It will be normalized to a url-safe
          form; changing it updates the address.
        </p>
        {errors.slug ? (
          <p className="text-sm text-destructive">{errors.slug.message}</p>
        ) : (
          <FieldError error={updateWorkspace.error} name="slug" />
        )}
      </div>

      <Button
        type="submit"
        disabled={updateWorkspace.isPending || !isDirty}
      >
        {updateWorkspace.isPending ? 'Saving…' : 'Save changes'}
      </Button>
    </form>
  )
}
