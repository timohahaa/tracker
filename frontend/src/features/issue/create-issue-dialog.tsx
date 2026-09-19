import { zodResolver } from '@hookform/resolvers/zod'
import { Plus } from 'lucide-react'
import { useMemo, useState } from 'react'
import { Controller, useForm } from 'react-hook-form'
import { useNavigate } from 'react-router-dom'
import { toast } from 'sonner'
import { z } from 'zod'

import { AppError } from '@/api/client'
import { FieldError } from '@/components/common/field-error'
import { PriorityBadge } from '@/components/priority-icon'
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
  SelectGroup,
  SelectItem,
  SelectLabel,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { Textarea } from '@/components/ui/textarea'
import { useCurrentWorkspace } from '@/features/auth/workspace-context'
import { useProjectsList } from '@/features/project/use-projects-list'
import { useTeamStatuses } from '@/features/team/use-team-statuses'
import { useTeams } from '@/features/team/use-teams'
import { type Priority,PRIORITY_ORDER } from '@/lib/priority'
import { STATUS_GROUP_ORDER, STATUS_GROUPS, type StatusGroup } from '@/lib/status'
import { AuthErrorBanner } from '@/pages/auth-shell'

import { useCreateIssue } from './use-issue'

// Sentinel for the "no project" choice, since Radix Select forbids empty-string
// item values.
const NO_PROJECT = '__none__'

// Client-side shape only; the backend validates team membership, status/project
// belonging and required fields. Server `fields[]` surface via <FieldError>.
const createIssueSchema = z.object({
  title: z.string().min(1, 'Title is required'),
  team_id: z.string().min(1, 'Team is required'),
  description: z.string(),
  priority: z.enum(['none', 'urgent', 'high', 'medium', 'low']),
  status_id: z.string(),
  project_id: z.string(),
  due_date: z.string(),
})

type CreateIssueValues = z.infer<typeof createIssueSchema>

const DEFAULTS: CreateIssueValues = {
  title: '',
  team_id: '',
  description: '',
  priority: 'none',
  status_id: '',
  project_id: '',
  due_date: '',
}

// "New issue" dialog. Team is required and drives the available statuses; status
// defaults to the team's backlog status on the backend when left unset. On
// success navigates to the created issue's detail page.
export function CreateIssueDialog({ defaultTeamId }: { defaultTeamId?: string }) {
  const { id: workspaceID, slug } = useCurrentWorkspace()
  const navigate = useNavigate()
  const createIssue = useCreateIssue()
  const [open, setOpen] = useState(false)

  const teamsQuery = useTeams(workspaceID)
  const projectsQuery = useProjectsList(workspaceID)
  const teams = teamsQuery.data?.data ?? []
  const projects = projectsQuery.data?.data ?? []

  const {
    register,
    handleSubmit,
    control,
    watch,
    setValue,
    reset,
    formState: { errors },
  } = useForm<CreateIssueValues>({
    resolver: zodResolver(createIssueSchema),
    defaultValues: { ...DEFAULTS, team_id: defaultTeamId ?? '' },
  })

  const teamId = watch('team_id')
  const statusesQuery = useTeamStatuses(workspaceID, teamId)
  const statuses = useMemo(
    () => (teamId ? (statusesQuery.data?.data ?? []) : []),
    [teamId, statusesQuery.data],
  )

  // Group the chosen team's statuses under their status-group headers.
  const statusSections = useMemo(() => {
    const buckets = new Map<StatusGroup, typeof statuses>()
    for (const status of statuses) {
      const group = (status.group as StatusGroup) ?? 'backlog'
      const bucket = buckets.get(group)
      if (bucket) bucket.push(status)
      else buckets.set(group, [status])
    }
    return STATUS_GROUP_ORDER.map((group) => ({
      group,
      items: buckets.get(group) ?? [],
    })).filter((section) => section.items.length > 0)
  }, [statuses])

  const handleOpenChange = (next: boolean) => {
    setOpen(next)
    if (!next) {
      reset({ ...DEFAULTS, team_id: defaultTeamId ?? '' })
      createIssue.reset()
    }
  }

  const onSubmit = (values: CreateIssueValues) => {
    createIssue.mutate(
      {
        params: { path: { workspaceID } },
        body: {
          title: values.title,
          team_id: values.team_id,
          description: values.description || undefined,
          priority: values.priority,
          status_id: values.status_id || undefined,
          project_id: values.project_id || undefined,
          due_date: values.due_date || undefined,
        },
      },
      {
        onSuccess: (data) => {
          toast.success('Issue created')
          handleOpenChange(false)
          const key = data.identifier || data.id
          if (key) navigate(`/${slug}/issues/${key}`)
        },
      },
    )
  }

  const serverError =
    createIssue.error instanceof AppError ? createIssue.error : null
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
          New issue
        </Button>
      </DialogTrigger>
      <DialogContent className="max-h-[90svh] overflow-y-auto sm:max-w-lg">
        <form onSubmit={handleSubmit(onSubmit)} noValidate>
          <DialogHeader>
            <DialogTitle>New issue</DialogTitle>
            <DialogDescription>
              Pick a team, give it a title, and optionally set status, priority,
              project and a due date.
            </DialogDescription>
          </DialogHeader>

          <div className="space-y-4 py-4">
            {bannerMessage ? <AuthErrorBanner message={bannerMessage} /> : null}

            <div className="space-y-1.5">
              <label htmlFor="issue-team" className="text-sm font-medium">
                Team
              </label>
              <Controller
                control={control}
                name="team_id"
                render={({ field }) => (
                  <Select
                    value={field.value || undefined}
                    onValueChange={(next) => {
                      field.onChange(next)
                      // Statuses are team-scoped; the previous pick no longer applies.
                      setValue('status_id', '')
                    }}
                  >
                    <SelectTrigger id="issue-team" aria-invalid={!!errors.team_id}>
                      <SelectValue placeholder="Select a team" />
                    </SelectTrigger>
                    <SelectContent>
                      {teams.map((team) => (
                        <SelectItem key={team.id} value={team.id ?? ''}>
                          {team.name || team.key || 'Team'}
                        </SelectItem>
                      ))}
                    </SelectContent>
                  </Select>
                )}
              />
              {errors.team_id ? (
                <p className="text-sm text-destructive">
                  {errors.team_id.message}
                </p>
              ) : (
                <FieldError error={createIssue.error} name="team_id" />
              )}
            </div>

            <div className="space-y-1.5">
              <label htmlFor="issue-title" className="text-sm font-medium">
                Title
              </label>
              <Input
                id="issue-title"
                placeholder="Issue title"
                aria-invalid={!!errors.title}
                {...register('title')}
              />
              {errors.title ? (
                <p className="text-sm text-destructive">
                  {errors.title.message}
                </p>
              ) : (
                <FieldError error={createIssue.error} name="title" />
              )}
            </div>

            <div className="space-y-1.5">
              <label htmlFor="issue-description" className="text-sm font-medium">
                Description
                <span className="ml-1 font-normal text-muted-foreground">
                  (optional)
                </span>
              </label>
              <Textarea
                id="issue-description"
                placeholder="Add more detail…"
                rows={4}
                {...register('description')}
              />
              <FieldError error={createIssue.error} name="description" />
            </div>

            <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
              <div className="space-y-1.5">
                <label className="text-sm font-medium">Priority</label>
                <Controller
                  control={control}
                  name="priority"
                  render={({ field }) => (
                    <Select
                      value={field.value}
                      onValueChange={(next) =>
                        field.onChange(next as Priority)
                      }
                    >
                      <SelectTrigger>
                        <SelectValue />
                      </SelectTrigger>
                      <SelectContent>
                        {PRIORITY_ORDER.map((priority) => (
                          <SelectItem key={priority} value={priority}>
                            <PriorityBadge priority={priority} />
                          </SelectItem>
                        ))}
                      </SelectContent>
                    </Select>
                  )}
                />
              </div>

              <div className="space-y-1.5">
                <label className="text-sm font-medium">Status</label>
                <Controller
                  control={control}
                  name="status_id"
                  render={({ field }) => (
                    <Select
                      value={field.value || undefined}
                      onValueChange={field.onChange}
                      disabled={!teamId || statuses.length === 0}
                    >
                      <SelectTrigger>
                        <SelectValue
                          placeholder={teamId ? 'Team default' : 'Pick a team'}
                        />
                      </SelectTrigger>
                      <SelectContent>
                        {statusSections.map((section) => (
                          <SelectGroup key={section.group}>
                            <SelectLabel>
                              {STATUS_GROUPS[section.group].label}
                            </SelectLabel>
                            {section.items.map((status) => (
                              <SelectItem
                                key={status.id}
                                value={status.id ?? ''}
                              >
                                {status.name}
                              </SelectItem>
                            ))}
                          </SelectGroup>
                        ))}
                      </SelectContent>
                    </Select>
                  )}
                />
              </div>

              <div className="space-y-1.5">
                <label className="text-sm font-medium">Project</label>
                <Controller
                  control={control}
                  name="project_id"
                  render={({ field }) => (
                    <Select
                      value={field.value || NO_PROJECT}
                      onValueChange={(next) =>
                        field.onChange(next === NO_PROJECT ? '' : next)
                      }
                    >
                      <SelectTrigger>
                        <SelectValue />
                      </SelectTrigger>
                      <SelectContent>
                        <SelectItem value={NO_PROJECT}>No project</SelectItem>
                        {projects.map((project) => (
                          <SelectItem key={project.id} value={project.id ?? ''}>
                            {project.name || 'Project'}
                          </SelectItem>
                        ))}
                      </SelectContent>
                    </Select>
                  )}
                />
              </div>

              <div className="space-y-1.5">
                <label htmlFor="issue-due" className="text-sm font-medium">
                  Due date
                </label>
                <Input id="issue-due" type="date" {...register('due_date')} />
              </div>
            </div>
          </div>

          <DialogFooter>
            <Button
              type="button"
              variant="outline"
              onClick={() => handleOpenChange(false)}
              disabled={createIssue.isPending}
            >
              Cancel
            </Button>
            <Button type="submit" disabled={createIssue.isPending}>
              {createIssue.isPending ? 'Creating…' : 'Create issue'}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  )
}
