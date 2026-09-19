import { zodResolver } from '@hookform/resolvers/zod'
import { useState } from 'react'
import { Controller, useForm } from 'react-hook-form'
import { useNavigate } from 'react-router-dom'
import { toast } from 'sonner'
import { z } from 'zod'

import { AppError } from '@/api/client'
import { FieldError } from '@/components/common/field-error'
import { StatusBadge } from '@/components/status-badge'
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
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { Textarea } from '@/components/ui/textarea'
import { useCurrentWorkspace } from '@/features/auth/workspace-context'
import { useTeams } from '@/features/team/use-teams'
import { useMembers } from '@/features/workspace/use-members'
import { AuthErrorBanner } from '@/pages/auth-shell'

import { LeadSelect } from './lead-select'
import { PROJECT_STATUS_ORDER, toProjectStatus } from './project-status'
import { TeamsMultiSelect } from './teams-multi-select'
import {
  type ProjectStatus,
  useClearProjectLead,
  useDeleteProject,
  useUpdateProject,
} from './use-projects'
import type { Project } from './use-projects-list'

const settingsSchema = z.object({
  name: z.string().min(1, 'Name is required'),
  description: z.string(),
  status: z.enum(['planned', 'in_progress', 'completed', 'canceled']),
  lead_id: z.string(),
  team_ids: z.array(z.string()),
})

type SettingsValues = z.infer<typeof settingsSchema>

function toFormValues(project: Project): SettingsValues {
  return {
    name: project.name ?? '',
    description: project.description ?? '',
    status: toProjectStatus(project.status),
    lead_id: project.lead_id ?? '',
    team_ids: project.team_ids ?? [],
  }
}

function DeleteProjectDialog({ project }: { project: Project }) {
  const { id: workspaceID, slug } = useCurrentWorkspace()
  const navigate = useNavigate()
  const deleteProject = useDeleteProject()
  const [open, setOpen] = useState(false)

  const error =
    deleteProject.error instanceof AppError ? deleteProject.error : null

  const handleDelete = () => {
    deleteProject.mutate(
      { params: { path: { workspaceID, projectID: project.id ?? '' } } },
      {
        onSuccess: () => {
          toast.success('Project deleted')
          setOpen(false)
          navigate(`/${slug}/projects`)
        },
      },
    )
  }

  return (
    <Dialog open={open} onOpenChange={setOpen}>
      <Button variant="destructive" onClick={() => setOpen(true)}>
        Delete project
      </Button>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>Delete project</DialogTitle>
          <DialogDescription>
            Delete <span className="font-medium">{project.name}</span>? Issues
            keep existing but are unlinked from this project. This cannot be
            undone.
          </DialogDescription>
        </DialogHeader>
        {error ? <AuthErrorBanner message={error.message} /> : null}
        <DialogFooter>
          <Button
            variant="outline"
            onClick={() => setOpen(false)}
            disabled={deleteProject.isPending}
          >
            Cancel
          </Button>
          <Button
            variant="destructive"
            onClick={handleDelete}
            disabled={deleteProject.isPending}
          >
            {deleteProject.isPending ? 'Deleting…' : 'Delete project'}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}

interface ProjectSettingsFormProps {
  project: Project
  // Workspace owner/admin: can delete the project (all members can edit).
  canDelete: boolean
}

// Edit form for a project's name/description/status/lead/teams. `team_ids` uses
// replace-semantics. Clearing the lead can't go through PATCH (nullable FK), so
// selecting "No lead" fires the dedicated DELETE lead endpoint after the patch.
export function ProjectSettingsForm({
  project,
  canDelete,
}: ProjectSettingsFormProps) {
  const { id: workspaceID } = useCurrentWorkspace()
  const updateProject = useUpdateProject()
  const clearLead = useClearProjectLead()

  const teamsQuery = useTeams(workspaceID)
  const membersQuery = useMembers(workspaceID)
  const teams = teamsQuery.data?.data ?? []
  const members = membersQuery.data?.data ?? []

  const {
    register,
    handleSubmit,
    control,
    reset,
    formState: { errors, isDirty },
  } = useForm<SettingsValues>({
    resolver: zodResolver(settingsSchema),
    values: toFormValues(project),
  })

  const serverError =
    updateProject.error instanceof AppError ? updateProject.error : null
  const bannerMessage =
    serverError &&
    (serverError.code !== 'validation' || serverError.fields.length === 0)
      ? serverError.message
      : null

  const isPending = updateProject.isPending || clearLead.isPending
  const projectID = project.id ?? ''

  const onSubmit = (values: SettingsValues) => {
    const shouldClearLead = !values.lead_id && !!project.lead_id

    updateProject.mutate(
      {
        params: { path: { workspaceID, projectID } },
        body: {
          name: values.name,
          description: values.description,
          status: values.status,
          team_ids: values.team_ids,
          ...(values.lead_id ? { lead_id: values.lead_id } : {}),
        },
      },
      {
        onSuccess: (data) => {
          if (shouldClearLead) {
            clearLead.mutate(
              { params: { path: { workspaceID, projectID } } },
              {
                onSuccess: (cleared) => {
                  toast.success('Project settings saved')
                  reset(toFormValues(cleared))
                },
                onError: (err) =>
                  toast.error(
                    err instanceof AppError ? err.message : 'Failed to save',
                  ),
              },
            )
            return
          }
          toast.success('Project settings saved')
          reset(toFormValues(data))
        },
      },
    )
  }

  return (
    <div className="space-y-6">
      <form onSubmit={handleSubmit(onSubmit)} className="space-y-4" noValidate>
        {bannerMessage ? <AuthErrorBanner message={bannerMessage} /> : null}

        <div className="space-y-1.5">
          <label htmlFor="project-name" className="text-sm font-medium">
            Name
          </label>
          <Input
            id="project-name"
            aria-invalid={!!errors.name}
            {...register('name')}
          />
          {errors.name ? (
            <p className="text-sm text-destructive">{errors.name.message}</p>
          ) : (
            <FieldError error={updateProject.error} name="name" />
          )}
        </div>

        <div className="space-y-1.5">
          <label htmlFor="project-description" className="text-sm font-medium">
            Description
          </label>
          <Textarea
            id="project-description"
            rows={4}
            aria-invalid={!!errors.description}
            {...register('description')}
          />
          <FieldError error={updateProject.error} name="description" />
        </div>

        <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
          <div className="space-y-1.5">
            <label className="text-sm font-medium">Status</label>
            <Controller
              control={control}
              name="status"
              render={({ field }) => (
                <Select
                  value={field.value}
                  onValueChange={(next) => field.onChange(next as ProjectStatus)}
                >
                  <SelectTrigger>
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent>
                    {PROJECT_STATUS_ORDER.map((status) => (
                      <SelectItem key={status} value={status}>
                        <StatusBadge status={toProjectStatus(status)} />
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
              )}
            />
          </div>

          <div className="space-y-1.5">
            <label htmlFor="project-lead" className="text-sm font-medium">
              Lead
            </label>
            <Controller
              control={control}
              name="lead_id"
              render={({ field }) => (
                <LeadSelect
                  id="project-lead"
                  members={members}
                  value={field.value}
                  onChange={field.onChange}
                />
              )}
            />
          </div>
        </div>

        <div className="space-y-1.5">
          <label className="text-sm font-medium">Teams</label>
          <Controller
            control={control}
            name="team_ids"
            render={({ field }) => (
              <TeamsMultiSelect
                teams={teams}
                value={field.value}
                onChange={field.onChange}
              />
            )}
          />
          <p className="text-xs text-muted-foreground">
            Linking an issue to this project also adds its team automatically.
          </p>
          <FieldError error={updateProject.error} name="team_ids" />
        </div>

        <Button type="submit" disabled={isPending || !isDirty}>
          {isPending ? 'Saving…' : 'Save changes'}
        </Button>
      </form>

      {canDelete ? (
        <div className="space-y-2 border-t pt-6">
          <h3 className="text-sm font-medium">Danger zone</h3>
          <p className="text-sm text-muted-foreground">
            Deleting a project unlinks its issues; the issues themselves are kept.
          </p>
          <DeleteProjectDialog project={project} />
        </div>
      ) : null}
    </div>
  )
}
