import { Pencil, Tag, Trash2 } from 'lucide-react'
import { useState } from 'react'
import { toast } from 'sonner'

import { AppError } from '@/api/client'
import { EmptyState } from '@/components/common/empty-state'
import { ErrorState } from '@/components/common/error-state'
import { ListSkeleton } from '@/components/common/skeletons'
import { LabelChip } from '@/components/label-dot'
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
import { useCurrentWorkspace } from '@/features/auth/workspace-context'

import { LabelFormDialog } from './label-form-dialog'
import { type Label, useDeleteLabel, useLabels } from './use-labels'

const FALLBACK_COLOR = '#8a8f98'

function DeleteLabelDialog({ label }: { label: Label }) {
  const { id: workspaceID } = useCurrentWorkspace()
  const deleteLabel = useDeleteLabel()
  const [open, setOpen] = useState(false)

  const error = deleteLabel.error instanceof AppError ? deleteLabel.error : null

  const handleDelete = () => {
    deleteLabel.mutate(
      { params: { path: { workspaceID, labelID: label.id ?? '' } } },
      {
        onSuccess: () => {
          toast.success('Label deleted')
          setOpen(false)
        },
      },
    )
  }

  return (
    <Dialog open={open} onOpenChange={setOpen}>
      <DialogTrigger asChild>
        <Button
          variant="ghost"
          size="icon"
          className="h-8 w-8 text-muted-foreground hover:text-destructive"
          aria-label={`Delete ${label.name}`}
        >
          <Trash2 className="h-4 w-4" />
        </Button>
      </DialogTrigger>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>Delete label</DialogTitle>
          <DialogDescription>
            Delete <span className="font-medium">{label.name}</span>? It will be
            removed from every issue that currently uses it. This cannot be
            undone.
          </DialogDescription>
        </DialogHeader>
        {error ? (
          <p className="text-sm text-destructive">{error.message}</p>
        ) : null}
        <DialogFooter>
          <Button
            variant="outline"
            onClick={() => setOpen(false)}
            disabled={deleteLabel.isPending}
          >
            Cancel
          </Button>
          <Button
            variant="destructive"
            onClick={handleDelete}
            disabled={deleteLabel.isPending}
          >
            {deleteLabel.isPending ? 'Deleting…' : 'Delete'}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}

function LabelRow({ label, teamID }: { label: Label; teamID?: string }) {
  return (
    <div className="flex items-center gap-2 border-b py-2 pl-2 last:border-b-0">
      <LabelChip
        name={label.name ?? ''}
        color={label.color ?? FALLBACK_COLOR}
        filled
      />
      <span className="min-w-0 flex-1 truncate font-mono text-xs text-muted-foreground">
        {label.color}
      </span>
      <div className="flex shrink-0 items-center gap-0.5">
        <LabelFormDialog
          label={label}
          teamID={teamID}
          trigger={
            <Button
              variant="ghost"
              size="icon"
              className="h-8 w-8 text-muted-foreground"
              aria-label={`Edit ${label.name}`}
            >
              <Pencil className="h-4 w-4" />
            </Button>
          }
        />
        <DeleteLabelDialog label={label} />
      </div>
    </div>
  )
}

interface LabelsManagerProps {
  // Scope: a team id → that team's labels; omitted → workspace-level labels
  // (`team_id` null/absent).
  teamID?: string
}

// Labels CRUD for one scope. Any workspace member can create/edit/delete labels
// (per the API); no role gating, but a 403 still surfaces via the dialogs.
export function LabelsManager({ teamID }: LabelsManagerProps) {
  const { id: workspaceID } = useCurrentWorkspace()
  const query = useLabels(workspaceID)

  if (query.isLoading) return <ListSkeleton rows={4} />
  if (query.isError) {
    return <ErrorState error={query.error} onRetry={() => query.refetch()} />
  }

  const all = query.data?.data ?? []
  const labels = teamID
    ? all.filter((l) => l.team_id === teamID)
    : all.filter((l) => !l.team_id)

  const createTrigger = (
    <Button variant="outline" size="sm">
      New label
    </Button>
  )

  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between gap-3">
        <p className="text-sm text-muted-foreground">
          {teamID
            ? 'Labels scoped to this team, usable on its issues.'
            : 'Workspace labels are available across every team.'}
        </p>
        <LabelFormDialog teamID={teamID} trigger={createTrigger} />
      </div>

      {labels.length === 0 ? (
        <EmptyState
          icon={Tag}
          title="No labels yet"
          description={
            teamID
              ? 'Create a label to categorize this team’s issues.'
              : 'Create a workspace label to categorize issues across teams.'
          }
          action={<LabelFormDialog teamID={teamID} trigger={createTrigger} />}
        />
      ) : (
        <div className="rounded-lg border px-2">
          {labels.map((label) => (
            <LabelRow key={label.id} label={label} teamID={teamID} />
          ))}
        </div>
      )}
    </div>
  )
}
