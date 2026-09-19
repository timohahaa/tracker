import { zodResolver } from '@hookform/resolvers/zod'
import { useState } from 'react'
import { useForm } from 'react-hook-form'
import { useNavigate } from 'react-router-dom'
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
} from '@/components/ui/dialog'
import { Input } from '@/components/ui/input'
import { Textarea } from '@/components/ui/textarea'
import { useCurrentWorkspace } from '@/features/auth/workspace-context'
import { AuthErrorBanner } from '@/pages/auth-shell'

import { type Team, useDeleteTeam, useUpdateTeam } from './use-teams'

const settingsSchema = z.object({
  name: z.string().min(1, 'Name is required'),
  key: z.string().min(1, 'Key is required'),
  description: z.string(),
})

type SettingsValues = z.infer<typeof settingsSchema>

interface TeamSettingsFormProps {
  team: Team
  // Team member: can edit team settings. Members get a read-only view.
  canManage: boolean
  // Workspace owner/admin: can delete the team.
  canDelete: boolean
}

function DeleteTeamDialog({ team }: { team: Team }) {
  const { id: workspaceID, slug } = useCurrentWorkspace()
  const navigate = useNavigate()
  const deleteTeam = useDeleteTeam()
  const [open, setOpen] = useState(false)

  const error = deleteTeam.error instanceof AppError ? deleteTeam.error : null

  const handleDelete = () => {
    deleteTeam.mutate(
      { params: { path: { workspaceID, teamID: team.id ?? '' } } },
      {
        onSuccess: () => {
          toast.success('Team deleted')
          setOpen(false)
          navigate(`/${slug}/teams`)
        },
      },
    )
  }

  return (
    <Dialog open={open} onOpenChange={setOpen}>
      <Button variant="destructive" onClick={() => setOpen(true)}>
        Delete team
      </Button>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>Delete team</DialogTitle>
          <DialogDescription>
            Delete <span className="font-medium">{team.name}</span>? Its issues,
            statuses and memberships will be removed. This cannot be undone.
          </DialogDescription>
        </DialogHeader>
        {error ? <AuthErrorBanner message={error.message} /> : null}
        <DialogFooter>
          <Button
            variant="outline"
            onClick={() => setOpen(false)}
            disabled={deleteTeam.isPending}
          >
            Cancel
          </Button>
          <Button
            variant="destructive"
            onClick={handleDelete}
            disabled={deleteTeam.isPending}
          >
            {deleteTeam.isPending ? 'Deleting…' : 'Delete team'}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}

export function TeamSettingsForm({
  team,
  canManage,
  canDelete,
}: TeamSettingsFormProps) {
  const { id: workspaceID } = useCurrentWorkspace()
  const updateTeam = useUpdateTeam()

  const {
    register,
    handleSubmit,
    reset,
    formState: { errors, isDirty },
  } = useForm<SettingsValues>({
    resolver: zodResolver(settingsSchema),
    values: {
      name: team.name ?? '',
      key: team.key ?? '',
      description: team.description ?? '',
    },
  })

  const serverError =
    updateTeam.error instanceof AppError ? updateTeam.error : null
  const bannerMessage =
    serverError &&
    (serverError.code !== 'validation' || serverError.fields.length === 0)
      ? serverError.message
      : null

  const onSubmit = (values: SettingsValues) => {
    updateTeam.mutate(
      {
        params: { path: { workspaceID, teamID: team.id ?? '' } },
        body: {
          name: values.name,
          key: values.key,
          description: values.description,
        },
      },
      {
        onSuccess: (data) => {
          toast.success('Team settings saved')
          reset({
            name: data.name ?? '',
            key: data.key ?? '',
            description: data.description ?? '',
          })
        },
      },
    )
  }

  if (!canManage) {
    return (
      <div className="space-y-4">
        <p className="text-sm text-muted-foreground">
          Only team members can change these settings.
        </p>
        <dl className="space-y-4">
          <div className="space-y-1">
            <dt className="text-sm font-medium">Name</dt>
            <dd className="text-sm text-muted-foreground">
              {team.name || '—'}
            </dd>
          </div>
          <div className="space-y-1">
            <dt className="text-sm font-medium">Key</dt>
            <dd className="font-mono text-sm text-muted-foreground">
              {team.key || '—'}
            </dd>
          </div>
          <div className="space-y-1">
            <dt className="text-sm font-medium">Description</dt>
            <dd className="text-sm text-muted-foreground">
              {team.description || '—'}
            </dd>
          </div>
        </dl>
        {canDelete ? (
          <div className="border-t pt-4">
            <DeleteTeamDialog team={team} />
          </div>
        ) : null}
      </div>
    )
  }

  return (
    <div className="space-y-6">
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
            <FieldError error={updateTeam.error} name="name" />
          )}
        </div>

        <div className="space-y-1.5">
          <label htmlFor="key" className="text-sm font-medium">
            Key
          </label>
          <Input
            id="key"
            className="font-mono uppercase"
            aria-invalid={!!errors.key}
            {...register('key')}
          />
          <p className="text-xs text-muted-foreground">
            Prefixes issue identifiers. Changing it affects new issue keys.
          </p>
          {errors.key ? (
            <p className="text-sm text-destructive">{errors.key.message}</p>
          ) : (
            <FieldError error={updateTeam.error} name="key" />
          )}
        </div>

        <div className="space-y-1.5">
          <label htmlFor="description" className="text-sm font-medium">
            Description
          </label>
          <Textarea
            id="description"
            aria-invalid={!!errors.description}
            {...register('description')}
          />
          <FieldError error={updateTeam.error} name="description" />
        </div>

        <Button type="submit" disabled={updateTeam.isPending || !isDirty}>
          {updateTeam.isPending ? 'Saving…' : 'Save changes'}
        </Button>
      </form>

      {canDelete ? (
        <div className="space-y-2 border-t pt-6">
          <h3 className="text-sm font-medium">Danger zone</h3>
          <p className="text-sm text-muted-foreground">
            Deleting a team removes its issues, statuses and memberships.
          </p>
          <DeleteTeamDialog team={team} />
        </div>
      ) : null}
    </div>
  )
}
