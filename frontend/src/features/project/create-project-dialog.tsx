import { zodResolver } from '@hookform/resolvers/zod'
import { Plus } from 'lucide-react'
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
  DialogTrigger,
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
import { type ProjectStatus, useCreateProject } from './use-projects'

// Client-side shape only; the backend validates the rest. Server `fields[]`
// surface via <FieldError>.
const createProjectSchema = z.object({
  name: z.string().min(1, 'Name is required'),
  description: z.string(),
  status: z.enum(['planned', 'in_progress', 'completed', 'canceled']),
  lead_id: z.string(),
  team_ids: z.array(z.string()),
})

type CreateProjectValues = z.infer<typeof createProjectSchema>

const DEFAULTS: CreateProjectValues = {
  name: '',
  description: '',
  status: 'planned',
  lead_id: '',
  team_ids: [],
}

// "New project" dialog. Available to any workspace member. On success navigates
// to the created project's detail page.
export function CreateProjectDialog() {
  const { id: workspaceID, slug } = useCurrentWorkspace()
  const navigate = useNavigate()
  const createProject = useCreateProject()
  const [open, setOpen] = useState(false)

  const teamsQuery = useTeams(workspaceID)
  const membersQuery = useMembers(workspaceID)
  const teams = teamsQuery.data?.data ?? []
  const members = membersQuery.data?.data ?? []

  const {
    register,
    handleSubmit,
    control,
    reset,
    formState: { errors },
  } = useForm<CreateProjectValues>({
    resolver: zodResolver(createProjectSchema),
    defaultValues: DEFAULTS,
  })

  const handleOpenChange = (next: boolean) => {
    setOpen(next)
    if (!next) {
      reset(DEFAULTS)
      createProject.reset()
    }
  }

  const onSubmit = (values: CreateProjectValues) => {
    createProject.mutate(
      {
        params: { path: { workspaceID } },
        body: {
          name: values.name,
          description: values.description || undefined,
          status: values.status,
          lead_id: values.lead_id || undefined,
          team_ids: values.team_ids.length > 0 ? values.team_ids : undefined,
        },
      },
      {
        onSuccess: (data) => {
          toast.success('Project created')
          handleOpenChange(false)
          if (data.id) navigate(`/${slug}/projects/${data.id}`)
        },
      },
    )
  }

  const serverError =
    createProject.error instanceof AppError ? createProject.error : null
  const bannerMessage =
    serverError &&
    (serverError.code !== 'validation' || serverError.fields.length === 0)
      ? serverError.message
      : null

  return (
    <Dialog open={open} onOpenChange={handleOpenChange}>
      <DialogTrigger asChild>
        <Button>
          <Plus className="h-4 w-4" />
          New project
        </Button>
      </DialogTrigger>
      <DialogContent className="max-h-[90svh] overflow-y-auto sm:max-w-lg">
        <form onSubmit={handleSubmit(onSubmit)} noValidate>
          <DialogHeader>
            <DialogTitle>New project</DialogTitle>
            <DialogDescription>
              Projects span one or more teams and group related issues.
            </DialogDescription>
          </DialogHeader>

          <div className="space-y-4 py-4">
            {bannerMessage ? <AuthErrorBanner message={bannerMessage} /> : null}

            <div className="space-y-1.5">
              <label htmlFor="project-name" className="text-sm font-medium">
                Name
              </label>
              <Input
                id="project-name"
                placeholder="Project name"
                aria-invalid={!!errors.name}
                {...register('name')}
              />
              {errors.name ? (
                <p className="text-sm text-destructive">
                  {errors.name.message}
                </p>
              ) : (
                <FieldError error={createProject.error} name="name" />
              )}
            </div>

            <div className="space-y-1.5">
              <label
                htmlFor="project-description"
                className="text-sm font-medium"
              >
                Description
                <span className="ml-1 font-normal text-muted-foreground">
                  (optional)
                </span>
              </label>
              <Textarea
                id="project-description"
                placeholder="What this project is about…"
                rows={3}
                {...register('description')}
              />
              <FieldError error={createProject.error} name="description" />
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
                      onValueChange={(next) =>
                        field.onChange(next as ProjectStatus)
                      }
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
              <label className="text-sm font-medium">
                Teams
                <span className="ml-1 font-normal text-muted-foreground">
                  (optional)
                </span>
              </label>
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
              <FieldError error={createProject.error} name="team_ids" />
            </div>
          </div>

          <DialogFooter>
            <Button
              type="button"
              variant="outline"
              onClick={() => handleOpenChange(false)}
              disabled={createProject.isPending}
            >
              Cancel
            </Button>
            <Button type="submit" disabled={createProject.isPending}>
              {createProject.isPending ? 'Creating…' : 'Create project'}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  )
}
