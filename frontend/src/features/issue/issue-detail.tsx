import { useQueryClient } from '@tanstack/react-query'
import { CalendarDays, Trash2 } from 'lucide-react'
import { useMemo, useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { toast } from 'sonner'

import { AppError } from '@/api/client'
import { PriorityBadge } from '@/components/priority-icon'
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
import { type Member, useMembers } from '@/features/workspace/use-members'
import { PRIORITIES, type Priority,PRIORITY_ORDER } from '@/lib/priority'
import { STATUS_GROUP_ORDER, STATUS_GROUPS, type StatusGroup } from '@/lib/status'

import { IssueAssignees } from './issue-assignees'
import { IssueLabels } from './issue-labels'
import {
  issueQueryKey,
  type UpdateIssueBody,
  useDeleteIssue,
  useUnlinkIssueProject,
  useUpdateIssue,
} from './use-issue'
import type { Issue } from './use-issues'

const NO_PROJECT = '__none__'

function errorMessage(err: unknown): string {
  return err instanceof AppError ? err.message : 'Something went wrong'
}

function toPriority(value: string | undefined): Priority {
  return value && value in PRIORITIES ? (value as Priority) : 'none'
}

function formatDate(iso: string | undefined): string | null {
  if (!iso) return null
  const date = new Date(iso)
  if (Number.isNaN(date.getTime())) return null
  return new Intl.DateTimeFormat(undefined, {
    year: 'numeric',
    month: 'short',
    day: 'numeric',
  }).format(date)
}

// A labeled row in the properties sidebar.
function PropertyRow({
  label,
  children,
}: {
  label: string
  children: React.ReactNode
}) {
  return (
    <div className="space-y-1.5">
      <span className="text-xs font-medium uppercase tracking-wide text-muted-foreground">
        {label}
      </span>
      <div>{children}</div>
    </div>
  )
}

interface IssueDetailProps {
  // The issue ref from the URL (identifier or UUID). Used as the mutation/cache
  // key so writes target the same query entry `useIssue` reads.
  issueID: string
  issue: Issue
}

// Full issue view with inline editing. Title/description/project/due-date PATCH
// on commit; status/priority use optimistic updates (instant, rolled back on
// error). Project clearing goes through the dedicated DELETE endpoint. Assignees
// and labels are shown read-only here (editing arrives in 2.4.3).
export function IssueDetail({ issueID, issue }: IssueDetailProps) {
  const { id: workspaceID, slug } = useCurrentWorkspace()
  const navigate = useNavigate()
  const queryClient = useQueryClient()

  const updateIssue = useUpdateIssue()
  const unlinkProject = useUnlinkIssueProject()

  const teamID = issue.team_id ?? ''
  const statusesQuery = useTeamStatuses(workspaceID, teamID)
  const statuses = useMemo(
    () => statusesQuery.data?.data ?? [],
    [statusesQuery.data],
  )
  const projectsQuery = useProjectsList(workspaceID)
  const projects = projectsQuery.data?.data ?? []
  const membersQuery = useMembers(workspaceID)
  const memberMap = useMemo(() => {
    const map = new Map<string, Member>()
    for (const member of membersQuery.data?.data ?? []) {
      if (member.user_id) map.set(member.user_id, member)
    }
    return map
  }, [membersQuery.data])

  const currentStatus = statuses.find((s) => s.id === issue.status_id)
  const currentGroup: StatusGroup =
    (currentStatus?.group as StatusGroup) ?? 'backlog'
  const priority = toPriority(issue.priority)

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

  // Optimistic single-field patch for status/priority: cancel in-flight refetches
  // (so a slower earlier request can't clobber the optimistic value), write the
  // cache immediately, then roll back on error. The mutation's onSuccess
  // reconciles with the server response. Only scalar fields are patched here, so
  // merging the request body onto the cached issue is safe (cast bridges the
  // request/response DTO shapes).
  const optimisticPatch = async (patch: UpdateIssueBody) => {
    const key = issueQueryKey(workspaceID, issueID)
    await queryClient.cancelQueries({ queryKey: key })
    const previous = queryClient.getQueryData<Issue>(key)
    queryClient.setQueryData<Issue>(key, (old) =>
      old ? ({ ...old, ...patch } as Issue) : old,
    )
    updateIssue.mutate(
      { params: { path: { workspaceID, issueID } }, body: patch },
      {
        onError: (err) => {
          queryClient.setQueryData(key, previous)
          toast.error(errorMessage(err))
        },
      },
    )
  }

  const commitPatch = (
    patch: UpdateIssueBody,
    { onSuccess }: { onSuccess?: () => void } = {},
  ) => {
    updateIssue.mutate(
      { params: { path: { workspaceID, issueID } }, body: patch },
      {
        onSuccess: () => onSuccess?.(),
        onError: (err) => toast.error(errorMessage(err)),
      },
    )
  }

  // --- Title (uncontrolled, remounts on server update via key) -------------
  const handleTitleBlur = (value: string) => {
    const next = value.trim()
    if (!next || next === issue.title) return
    commitPatch({ title: next })
  }

  // --- Description (edit mode) ---------------------------------------------
  const [editingDescription, setEditingDescription] = useState(false)
  const [descriptionDraft, setDescriptionDraft] = useState('')
  const openDescriptionEditor = () => {
    setDescriptionDraft(issue.description ?? '')
    setEditingDescription(true)
  }
  const saveDescription = () => {
    commitPatch(
      { description: descriptionDraft },
      { onSuccess: () => setEditingDescription(false) },
    )
  }

  // --- Project -------------------------------------------------------------
  const handleProjectChange = (next: string) => {
    if (next === NO_PROJECT) {
      unlinkProject.mutate(
        { params: { path: { workspaceID, issueID } } },
        { onError: (err) => toast.error(errorMessage(err)) },
      )
      return
    }
    commitPatch({ project_id: next })
  }

  // --- Due date (set/change only; the API has no clear path) ---------------
  const handleDueDateChange = (value: string) => {
    if (!value) {
      toast.info('Due date cannot be cleared')
      return
    }
    if (value === issue.due_date) return
    commitPatch({ due_date: value })
  }

  // --- Delete --------------------------------------------------------------
  const [deleteOpen, setDeleteOpen] = useState(false)
  const deleteIssue = useDeleteIssue()
  const handleDelete = () => {
    deleteIssue.mutate(
      { params: { path: { workspaceID, issueID } } },
      {
        onSuccess: () => {
          toast.success('Issue deleted')
          setDeleteOpen(false)
          navigate(`/${slug}/issues`)
        },
        onError: (err) => toast.error(errorMessage(err)),
      },
    )
  }

  const created = formatDate(issue.created_at)
  const updated = formatDate(issue.updated_at)

  return (
    <div className="grid gap-8 lg:grid-cols-[1fr_260px]">
      {/* Main column */}
      <div className="min-w-0 space-y-6">
        <div className="flex items-center gap-3">
          <span className="font-mono text-sm text-muted-foreground">
            {issue.identifier}
          </span>
        </div>

        <Input
          key={`${issue.title}-${issue.updated_at}`}
          defaultValue={issue.title}
          aria-label="Issue title"
          onBlur={(event) => handleTitleBlur(event.target.value)}
          onKeyDown={(event) => {
            if (event.key === 'Enter') event.currentTarget.blur()
          }}
          className="h-auto border-none px-0 text-2xl font-semibold shadow-none focus-visible:ring-0"
        />

        <div className="space-y-2">
          <span className="text-xs font-medium uppercase tracking-wide text-muted-foreground">
            Description
          </span>
          {editingDescription ? (
            <div className="space-y-2">
              <Textarea
                value={descriptionDraft}
                onChange={(event) => setDescriptionDraft(event.target.value)}
                rows={8}
                autoFocus
                placeholder="Add a description…"
              />
              <div className="flex gap-2">
                <Button
                  size="sm"
                  onClick={saveDescription}
                  disabled={updateIssue.isPending}
                >
                  {updateIssue.isPending ? 'Saving…' : 'Save'}
                </Button>
                <Button
                  size="sm"
                  variant="outline"
                  onClick={() => setEditingDescription(false)}
                  disabled={updateIssue.isPending}
                >
                  Cancel
                </Button>
              </div>
            </div>
          ) : issue.description ? (
            <button
              type="button"
              onClick={openDescriptionEditor}
              className="block w-full whitespace-pre-wrap rounded-md p-3 text-left text-sm hover:bg-accent/50"
            >
              {issue.description}
            </button>
          ) : (
            <button
              type="button"
              onClick={openDescriptionEditor}
              className="block w-full rounded-md p-3 text-left text-sm text-muted-foreground hover:bg-accent/50"
            >
              Add a description…
            </button>
          )}
        </div>
      </div>

      {/* Properties sidebar */}
      <aside className="space-y-5 lg:border-l lg:pl-6">
        <PropertyRow label="Status">
          <Select
            value={issue.status_id || undefined}
            onValueChange={(next) => {
              void optimisticPatch({ status_id: next })
            }}
            disabled={statuses.length === 0}
          >
            <SelectTrigger className="w-full">
              <SelectValue placeholder="Status">
                <span className="flex items-center gap-1.5">
                  <StatusBadge status={currentGroup} iconOnly />
                  <span className="truncate">
                    {currentStatus?.name ?? 'No status'}
                  </span>
                </span>
              </SelectValue>
            </SelectTrigger>
            <SelectContent>
              {statusSections.map((section) => (
                <SelectGroup key={section.group}>
                  <SelectLabel>{STATUS_GROUPS[section.group].label}</SelectLabel>
                  {section.items.map((status) => (
                    <SelectItem key={status.id} value={status.id ?? ''}>
                      {status.name}
                    </SelectItem>
                  ))}
                </SelectGroup>
              ))}
            </SelectContent>
          </Select>
        </PropertyRow>

        <PropertyRow label="Priority">
          <Select
            value={priority}
            onValueChange={(next) => {
              void optimisticPatch({ priority: next as Priority })
            }}
          >
            <SelectTrigger className="w-full">
              <SelectValue>
                <PriorityBadge priority={priority} />
              </SelectValue>
            </SelectTrigger>
            <SelectContent>
              {PRIORITY_ORDER.map((value) => (
                <SelectItem key={value} value={value}>
                  <PriorityBadge priority={value} />
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
        </PropertyRow>

        <PropertyRow label="Project">
          <Select
            value={issue.project_id || NO_PROJECT}
            onValueChange={handleProjectChange}
            disabled={unlinkProject.isPending}
          >
            <SelectTrigger className="w-full">
              <SelectValue placeholder="No project" />
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
        </PropertyRow>

        <PropertyRow label="Due date">
          <Input
            type="date"
            value={issue.due_date ?? ''}
            onChange={(event) => handleDueDateChange(event.target.value)}
            className="w-full"
          />
        </PropertyRow>

        <PropertyRow label="Assignees">
          <IssueAssignees
            workspaceID={workspaceID}
            issueID={issueID}
            issue={issue}
            memberMap={memberMap}
          />
        </PropertyRow>

        <PropertyRow label="Labels">
          <IssueLabels
            workspaceID={workspaceID}
            issueID={issueID}
            issue={issue}
          />
        </PropertyRow>

        <div className="space-y-1.5 border-t pt-4 text-xs text-muted-foreground">
          {created ? (
            <p className="flex items-center gap-1.5">
              <CalendarDays className="h-3.5 w-3.5" aria-hidden="true" />
              Created {created}
            </p>
          ) : null}
          {updated ? <p>Updated {updated}</p> : null}
        </div>

        <div className="border-t pt-4">
          <Button
            variant="ghost"
            size="sm"
            className="text-destructive hover:text-destructive"
            onClick={() => setDeleteOpen(true)}
          >
            <Trash2 className="h-4 w-4" />
            Delete issue
          </Button>
        </div>
      </aside>

      <Dialog open={deleteOpen} onOpenChange={setDeleteOpen}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Delete issue</DialogTitle>
            <DialogDescription>
              Delete{' '}
              <span className="font-mono font-medium">{issue.identifier}</span>?
              This cannot be undone.
            </DialogDescription>
          </DialogHeader>
          <DialogFooter>
            <Button
              variant="outline"
              onClick={() => setDeleteOpen(false)}
              disabled={deleteIssue.isPending}
            >
              Cancel
            </Button>
            <Button
              variant="destructive"
              onClick={handleDelete}
              disabled={deleteIssue.isPending}
            >
              {deleteIssue.isPending ? 'Deleting…' : 'Delete issue'}
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  )
}
