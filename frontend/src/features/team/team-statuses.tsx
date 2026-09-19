import { ChevronDown, ChevronUp, Pencil, Trash2 } from 'lucide-react'
import { useState } from 'react'
import { toast } from 'sonner'

import { AppError } from '@/api/client'
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
import { useCurrentWorkspace } from '@/features/auth/workspace-context'
import { STATUS_GROUP_ORDER, type StatusGroup } from '@/lib/status'

import {
  type TeamStatus,
  useCreateStatus,
  useDeleteStatus,
  useUpdateStatus,
} from './use-team-statuses'

function toastError(error: unknown, fallback: string) {
  toast.error(error instanceof AppError ? error.message : fallback)
}

// Groups statuses by their `group` and sorts each group by position.
function groupStatuses(statuses: TeamStatus[]): Record<StatusGroup, TeamStatus[]> {
  const byGroup = {
    backlog: [],
    planned: [],
    in_progress: [],
    completed: [],
    canceled: [],
  } as Record<StatusGroup, TeamStatus[]>

  for (const status of statuses) {
    const group = status.group as StatusGroup
    if (group in byGroup) byGroup[group].push(status)
  }
  for (const group of STATUS_GROUP_ORDER) {
    byGroup[group].sort((a, b) => (a.position ?? 0) - (b.position ?? 0))
  }
  return byGroup
}

// Next free position in a group (positions are unique within a group).
function nextGroupPosition(list: TeamStatus[]): number {
  if (list.length === 0) return 0
  return Math.max(...list.map((s) => s.position ?? 0)) + 1
}

function RenameStatusDialog({
  status,
  teamID,
}: {
  status: TeamStatus
  teamID: string
}) {
  const { id: workspaceID } = useCurrentWorkspace()
  const updateStatus = useUpdateStatus()
  const [open, setOpen] = useState(false)
  const [name, setName] = useState(status.name ?? '')

  const handleOpenChange = (next: boolean) => {
    setOpen(next)
    if (next) setName(status.name ?? '')
  }

  const handleSave = () => {
    const trimmed = name.trim()
    if (!trimmed) return
    updateStatus.mutate(
      {
        params: { path: { workspaceID, teamID, statusID: status.id ?? '' } },
        body: { name: trimmed },
      },
      {
        onSuccess: () => {
          toast.success('Status renamed')
          setOpen(false)
        },
        onError: (error) => toastError(error, 'Failed to rename status'),
      },
    )
  }

  return (
    <Dialog open={open} onOpenChange={handleOpenChange}>
      <DialogTrigger asChild>
        <Button
          variant="ghost"
          size="icon"
          className="h-8 w-8 text-muted-foreground"
          aria-label={`Rename ${status.name}`}
        >
          <Pencil className="h-4 w-4" />
        </Button>
      </DialogTrigger>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>Rename status</DialogTitle>
          <DialogDescription>Update the status name.</DialogDescription>
        </DialogHeader>
        <Input
          value={name}
          onChange={(e) => setName(e.target.value)}
          aria-label="Status name"
          onKeyDown={(e) => {
            if (e.key === 'Enter') {
              e.preventDefault()
              handleSave()
            }
          }}
        />
        <DialogFooter>
          <Button
            variant="outline"
            onClick={() => setOpen(false)}
            disabled={updateStatus.isPending}
          >
            Cancel
          </Button>
          <Button
            onClick={handleSave}
            disabled={updateStatus.isPending || !name.trim()}
          >
            {updateStatus.isPending ? 'Saving…' : 'Save'}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}

function DeleteStatusButton({
  status,
  teamID,
  disabled,
}: {
  status: TeamStatus
  teamID: string
  // True when this is the only status in its group (backend forbids deletion).
  disabled: boolean
}) {
  const { id: workspaceID } = useCurrentWorkspace()
  const deleteStatus = useDeleteStatus()
  const [open, setOpen] = useState(false)

  const error = deleteStatus.error instanceof AppError ? deleteStatus.error : null

  const handleDelete = () => {
    deleteStatus.mutate(
      { params: { path: { workspaceID, teamID, statusID: status.id ?? '' } } },
      {
        onSuccess: () => {
          toast.success('Status deleted')
          setOpen(false)
        },
      },
    )
  }

  if (disabled) {
    return (
      <Button
        variant="ghost"
        size="icon"
        className="h-8 w-8 text-muted-foreground"
        disabled
        title="A group must keep at least one status"
        aria-label={`Delete ${status.name} (disabled: last status in group)`}
      >
        <Trash2 className="h-4 w-4" />
      </Button>
    )
  }

  return (
    <Dialog open={open} onOpenChange={setOpen}>
      <DialogTrigger asChild>
        <Button
          variant="ghost"
          size="icon"
          className="h-8 w-8 text-muted-foreground hover:text-destructive"
          aria-label={`Delete ${status.name}`}
        >
          <Trash2 className="h-4 w-4" />
        </Button>
      </DialogTrigger>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>Delete status</DialogTitle>
          <DialogDescription>
            Delete <span className="font-medium">{status.name}</span>? Issues
            using it must be moved to another status first.
          </DialogDescription>
        </DialogHeader>
        {error ? (
          <p className="text-sm text-destructive">{error.message}</p>
        ) : null}
        <DialogFooter>
          <Button
            variant="outline"
            onClick={() => setOpen(false)}
            disabled={deleteStatus.isPending}
          >
            Cancel
          </Button>
          <Button
            variant="destructive"
            onClick={handleDelete}
            disabled={deleteStatus.isPending}
          >
            {deleteStatus.isPending ? 'Deleting…' : 'Delete'}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}

function CreateStatusDialog({
  teamID,
  byGroup,
}: {
  teamID: string
  byGroup: Record<StatusGroup, TeamStatus[]>
}) {
  const { id: workspaceID } = useCurrentWorkspace()
  const createStatus = useCreateStatus()
  const [open, setOpen] = useState(false)
  const [group, setGroup] = useState<StatusGroup>('backlog')
  const [name, setName] = useState('')

  const handleOpenChange = (next: boolean) => {
    setOpen(next)
    if (next) {
      setGroup('backlog')
      setName('')
      createStatus.reset()
    }
  }

  const error = createStatus.error instanceof AppError ? createStatus.error : null

  const handleCreate = () => {
    const trimmed = name.trim()
    if (!trimmed) return
    // Position is unique within a group (backend 409s on collision), so append
    // the new status at the end of its group.
    const nextPosition = nextGroupPosition(byGroup[group])
    createStatus.mutate(
      {
        params: { path: { workspaceID, teamID } },
        body: { group, name: trimmed, position: nextPosition },
      },
      {
        onSuccess: () => {
          toast.success('Status created')
          handleOpenChange(false)
        },
      },
    )
  }

  return (
    <Dialog open={open} onOpenChange={handleOpenChange}>
      <DialogTrigger asChild>
        <Button variant="outline" size="sm">
          New status
        </Button>
      </DialogTrigger>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>Create status</DialogTitle>
          <DialogDescription>
            Add a status to one of the workflow groups.
          </DialogDescription>
        </DialogHeader>
        <div className="space-y-4">
          {error ? (
            <p className="text-sm text-destructive">{error.message}</p>
          ) : null}
          <div className="space-y-1.5">
            <label className="text-sm font-medium">Group</label>
            <Select
              value={group}
              onValueChange={(v) => setGroup(v as StatusGroup)}
            >
              <SelectTrigger aria-label="Status group">
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                {STATUS_GROUP_ORDER.map((g) => (
                  <SelectItem key={g} value={g}>
                    <StatusBadge status={g} />
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          </div>
          <div className="space-y-1.5">
            <label htmlFor="new-status-name" className="text-sm font-medium">
              Name
            </label>
            <Input
              id="new-status-name"
              value={name}
              onChange={(e) => setName(e.target.value)}
              placeholder="In Review"
            />
          </div>
        </div>
        <DialogFooter>
          <Button
            variant="outline"
            onClick={() => handleOpenChange(false)}
            disabled={createStatus.isPending}
          >
            Cancel
          </Button>
          <Button
            onClick={handleCreate}
            disabled={createStatus.isPending || !name.trim()}
          >
            {createStatus.isPending ? 'Creating…' : 'Create'}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}

interface StatusRowProps {
  status: TeamStatus
  teamID: string
  canManage: boolean
  isFirst: boolean
  isLast: boolean
  isOnly: boolean
  onMove: (direction: 'up' | 'down') => void
  moving: boolean
}

function StatusRow({
  status,
  teamID,
  canManage,
  isFirst,
  isLast,
  isOnly,
  onMove,
  moving,
}: StatusRowProps) {
  return (
    <div className="flex items-center gap-2 border-b py-2 pl-2 last:border-b-0">
      <span className="min-w-0 flex-1 truncate text-sm">{status.name}</span>
      {canManage ? (
        <div className="flex shrink-0 items-center gap-0.5">
          <Button
            variant="ghost"
            size="icon"
            className="h-8 w-8 text-muted-foreground"
            disabled={isFirst || moving}
            onClick={() => onMove('up')}
            aria-label={`Move ${status.name} up`}
          >
            <ChevronUp className="h-4 w-4" />
          </Button>
          <Button
            variant="ghost"
            size="icon"
            className="h-8 w-8 text-muted-foreground"
            disabled={isLast || moving}
            onClick={() => onMove('down')}
            aria-label={`Move ${status.name} down`}
          >
            <ChevronDown className="h-4 w-4" />
          </Button>
          <RenameStatusDialog status={status} teamID={teamID} />
          <DeleteStatusButton
            status={status}
            teamID={teamID}
            disabled={isOnly}
          />
        </div>
      ) : null}
    </div>
  )
}

interface TeamStatusesProps {
  teamID: string
  statuses: TeamStatus[]
  canManage: boolean
}

export function TeamStatuses({ teamID, statuses, canManage }: TeamStatusesProps) {
  const { id: workspaceID } = useCurrentWorkspace()
  const updateStatus = useUpdateStatus()
  const byGroup = groupStatuses(statuses)

  // Simple reorder: swap the position of two adjacent statuses in a group.
  // Positions are unique within a group, so a naive two-write swap would collide
  // on the shared constraint. Park the moved status at a free temp position
  // first, then fill the two slots. Fancy drag-and-drop is deferred to Phase 3.
  const move = async (
    group: StatusGroup,
    index: number,
    direction: 'up' | 'down',
  ) => {
    const list = byGroup[group]
    const targetIndex = direction === 'up' ? index - 1 : index + 1
    const current = list[index]
    const target = list[targetIndex]
    if (!current?.id || !target?.id) return

    const currentId = current.id
    const targetId = target.id
    const currentPos = current.position ?? index
    const targetPos = target.position ?? targetIndex
    const tempPos = nextGroupPosition(list)

    const patch = (statusID: string, position: number) =>
      updateStatus.mutateAsync({
        params: { path: { workspaceID, teamID, statusID } },
        body: { position },
      })

    try {
      await patch(currentId, tempPos)
      await patch(targetId, currentPos)
      await patch(currentId, targetPos)
    } catch (error) {
      toastError(error, 'Failed to reorder status')
    }
  }

  return (
    <div className="space-y-6">
      {canManage ? (
        <div className="flex justify-end">
          <CreateStatusDialog teamID={teamID} byGroup={byGroup} />
        </div>
      ) : null}

      {STATUS_GROUP_ORDER.map((group) => {
        const list = byGroup[group]
        return (
          <section key={group} className="space-y-1">
            <div className="flex items-center gap-2 px-2">
              <StatusBadge status={group} />
              <span className="text-xs text-muted-foreground">
                {list.length}
              </span>
            </div>
            {list.length === 0 ? (
              <p className="px-2 py-2 text-sm text-muted-foreground">
                No statuses in this group.
              </p>
            ) : (
              <div className="rounded-lg border px-2">
                {list.map((status, index) => (
                  <StatusRow
                    key={status.id}
                    status={status}
                    teamID={teamID}
                    canManage={canManage}
                    isFirst={index === 0}
                    isLast={index === list.length - 1}
                    isOnly={list.length === 1}
                    moving={updateStatus.isPending}
                    onMove={(direction) => {
                      void move(group, index, direction)
                    }}
                  />
                ))}
              </div>
            )}
          </section>
        )
      })}
    </div>
  )
}
