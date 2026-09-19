import { useQueryClient } from '@tanstack/react-query'
import { Check, Tag } from 'lucide-react'
import { useMemo, useState } from 'react'
import { toast } from 'sonner'

import { AppError } from '@/api/client'
import { LabelChip, LabelDot } from '@/components/label-dot'
import { Button } from '@/components/ui/button'
import {
  Command,
  CommandEmpty,
  CommandGroup,
  CommandInput,
  CommandItem,
  CommandList,
} from '@/components/ui/command'
import {
  Popover,
  PopoverContent,
  PopoverTrigger,
} from '@/components/ui/popover'
import { useLabels } from '@/features/label/use-labels'

import { issueQueryKey, useUpdateIssue } from './use-issue'
import type { Issue } from './use-issues'

const FALLBACK_COLOR = '#8a8f98'

function errorMessage(err: unknown): string {
  return err instanceof AppError ? err.message : 'Something went wrong'
}

interface IssueLabelsProps {
  workspaceID: string
  // Issue ref from the URL (identifier or UUID) — the mutation/cache key.
  issueID: string
  issue: Issue
}

// Editable label set for an issue. Labels attach/detach only via `label_ids`
// replace on PATCH (no point-op endpoints). The picker is restricted to
// workspace-level labels plus the issue team's labels (a cross-team label is a
// 400). Toggles are optimistic with rollback + toast on error.
export function IssueLabels({ workspaceID, issueID, issue }: IssueLabelsProps) {
  const queryClient = useQueryClient()
  const labelsQuery = useLabels(workspaceID)
  const updateIssue = useUpdateIssue()
  const [pickerOpen, setPickerOpen] = useState(false)

  const teamID = issue.team_id ?? ''
  const selected = issue.labels ?? []

  // Valid picker options: workspace-level (no team_id) + the issue team's labels.
  const options = useMemo(() => {
    const all = labelsQuery.data?.data ?? []
    return all.filter((l) => !l.team_id || l.team_id === teamID)
  }, [labelsQuery.data, teamID])

  const selectedIds = useMemo(
    () => new Set((issue.labels ?? []).map((l) => l.id).filter(Boolean)),
    [issue.labels],
  )

  const toggleLabel = async (labelId: string) => {
    const isSelected = selectedIds.has(labelId)
    const currentIds = selected
      .map((l) => l.id)
      .filter((id): id is string => !!id)
    const nextIds = isSelected
      ? currentIds.filter((id) => id !== labelId)
      : [...currentIds, labelId]

    const key = issueQueryKey(workspaceID, issueID)
    await queryClient.cancelQueries({ queryKey: key })
    const previous = queryClient.getQueryData<Issue>(key)

    const byId = new Map(options.map((l) => [l.id, l]))
    const nextLabels = nextIds.map((id) => {
      const full = byId.get(id)
      return { id, name: full?.name, color: full?.color }
    })
    queryClient.setQueryData<Issue>(key, (old) =>
      old ? { ...old, labels: nextLabels } : old,
    )

    updateIssue.mutate(
      { params: { path: { workspaceID, issueID } }, body: { label_ids: nextIds } },
      {
        onError: (err) => {
          queryClient.setQueryData(key, previous)
          toast.error(errorMessage(err))
        },
      },
    )
  }

  return (
    <div className="space-y-2">
      {selected.length > 0 ? (
        <div className="flex flex-wrap gap-1.5">
          {selected.map((label) => (
            <LabelChip
              key={label.id}
              name={label.name ?? ''}
              color={label.color ?? FALLBACK_COLOR}
            />
          ))}
        </div>
      ) : (
        <span className="text-sm text-muted-foreground">No labels</span>
      )}

      <Popover open={pickerOpen} onOpenChange={setPickerOpen}>
        <PopoverTrigger asChild>
          <Button variant="outline" size="sm" className="w-full justify-start">
            <Tag className="h-3.5 w-3.5" />
            Labels
          </Button>
        </PopoverTrigger>
        <PopoverContent className="w-64 p-0" align="start">
          <Command>
            <CommandInput placeholder="Search labels…" />
            <CommandList>
              <CommandEmpty>
                {labelsQuery.isLoading ? 'Loading…' : 'No labels found.'}
              </CommandEmpty>
              <CommandGroup>
                {options.map((label) => {
                  const id = label.id ?? ''
                  if (!id) return null
                  const isSelected = selectedIds.has(id)
                  return (
                    <CommandItem
                      key={id}
                      value={`${label.name ?? ''} ${id}`}
                      onSelect={() => void toggleLabel(id)}
                      className="gap-2"
                    >
                      <LabelDot color={label.color ?? FALLBACK_COLOR} />
                      <span className="min-w-0 flex-1 truncate">
                        {label.name}
                      </span>
                      {isSelected ? (
                        <Check className="h-4 w-4 shrink-0" />
                      ) : null}
                    </CommandItem>
                  )
                })}
              </CommandGroup>
            </CommandList>
          </Command>
        </PopoverContent>
      </Popover>
    </div>
  )
}
