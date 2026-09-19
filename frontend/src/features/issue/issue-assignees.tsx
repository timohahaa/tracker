import { useQueryClient } from '@tanstack/react-query'
import { Check, Star, UserPlus } from 'lucide-react'
import { useMemo, useState } from 'react'
import { toast } from 'sonner'

import { AppError } from '@/api/client'
import { Avatar, AvatarFallback } from '@/components/ui/avatar'
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
import { useTeamMembers } from '@/features/team/use-team-members'
import type { Member } from '@/features/workspace/use-members'
import { cn } from '@/lib/utils'

import {
  issueQueryKey,
  useAddAssignee,
  useRemoveAssignee,
  useSetPrimaryAssignee,
  useUpdateIssue,
} from './use-issue'
import type { Issue } from './use-issues'

function errorMessage(err: unknown): string {
  return err instanceof AppError ? err.message : 'Something went wrong'
}

function memberInitials(member: Member | undefined): string {
  const source = member?.display_name?.trim() || member?.email || ''
  const parts = source.split(/\s+/).filter(Boolean)
  if (parts.length === 0) return '?'
  if (parts.length === 1) return parts[0].slice(0, 2).toUpperCase()
  return (parts[0][0] + parts[parts.length - 1][0]).toUpperCase()
}

function memberName(member: Member | undefined, userId: string): string {
  return member?.display_name?.trim() || member?.email || userId
}

interface IssueAssigneesProps {
  workspaceID: string
  // Issue ref from the URL (identifier or UUID) — the mutation/cache key.
  issueID: string
  issue: Issue
  // Workspace members keyed by user_id, for display info (team members carry no
  // name/email).
  memberMap: Map<string, Member>
}

// Editable assignee set for an issue. Add/remove use the point-op endpoints;
// the responsible (primary) is set via PUT .../primary and cleared via a bulk
// `assignees` PATCH. All edits are optimistic (handled in the hooks / inline)
// with rollback + toast on error. The picker is limited to the issue's team.
export function IssueAssignees({
  workspaceID,
  issueID,
  issue,
  memberMap,
}: IssueAssigneesProps) {
  const queryClient = useQueryClient()
  const teamID = issue.team_id ?? ''

  const teamMembersQuery = useTeamMembers(workspaceID, teamID)
  const teamMembers = useMemo(
    () => teamMembersQuery.data?.data ?? [],
    [teamMembersQuery.data],
  )

  const addAssignee = useAddAssignee()
  const removeAssignee = useRemoveAssignee()
  const setPrimary = useSetPrimaryAssignee()
  const updateIssue = useUpdateIssue()

  const [pickerOpen, setPickerOpen] = useState(false)

  const assignees = issue.assignees ?? []
  const assignedIds = useMemo(
    () => new Set((issue.assignees ?? []).map((a) => a.user_id).filter(Boolean)),
    [issue.assignees],
  )

  const toggleAssignee = (userId: string, isAssigned: boolean) => {
    if (isAssigned) {
      removeAssignee.mutate(
        { params: { path: { workspaceID, issueID, userID: userId } } },
        { onError: (err) => toast.error(errorMessage(err)) },
      )
    } else {
      addAssignee.mutate(
        {
          params: { path: { workspaceID, issueID } },
          body: { user_id: userId, is_primary: false },
        },
        { onError: (err) => toast.error(errorMessage(err)) },
      )
    }
  }

  const setResponsible = (userId: string) => {
    setPrimary.mutate(
      { params: { path: { workspaceID, issueID, userID: userId } } },
      { onError: (err) => toast.error(errorMessage(err)) },
    )
  }

  // Clear the responsible while keeping everyone assigned: bulk `assignees`
  // replace with is_primary=false. Optimistic with rollback on error.
  const clearResponsible = async () => {
    const key = issueQueryKey(workspaceID, issueID)
    await queryClient.cancelQueries({ queryKey: key })
    const previous = queryClient.getQueryData<Issue>(key)
    queryClient.setQueryData<Issue>(key, (old) =>
      old
        ? {
            ...old,
            assignees: (old.assignees ?? []).map((a) => ({
              ...a,
              is_primary: false,
            })),
          }
        : old,
    )
    const body = {
      assignees: (previous?.assignees ?? [])
        .map((a) => a.user_id)
        .filter((id): id is string => !!id)
        .map((id) => ({ user_id: id, is_primary: false })),
    }
    updateIssue.mutate(
      { params: { path: { workspaceID, issueID } }, body },
      {
        onError: (err) => {
          queryClient.setQueryData(key, previous)
          toast.error(errorMessage(err))
        },
      },
    )
  }

  const toggleResponsible = (userId: string, isPrimary: boolean) => {
    if (isPrimary) void clearResponsible()
    else setResponsible(userId)
  }

  return (
    <div className="space-y-2">
      {assignees.length > 0 ? (
        <ul className="space-y-1">
          {assignees.map((assignee) => {
            const userId = assignee.user_id ?? ''
            const member = memberMap.get(userId)
            const isPrimary = !!assignee.is_primary
            return (
              <li key={userId} className="flex items-center gap-2 text-sm">
                <Avatar
                  className={cn(
                    'h-5 w-5',
                    isPrimary && 'ring-1 ring-primary',
                  )}
                >
                  <AvatarFallback className="text-[10px]">
                    {memberInitials(member)}
                  </AvatarFallback>
                </Avatar>
                <span className="min-w-0 flex-1 truncate">
                  {memberName(member, userId)}
                </span>
                <button
                  type="button"
                  onClick={() => toggleResponsible(userId, isPrimary)}
                  className="shrink-0 rounded p-0.5 text-muted-foreground hover:text-foreground"
                  aria-label={
                    isPrimary ? 'Clear responsible' : 'Set as responsible'
                  }
                  aria-pressed={isPrimary}
                  title={isPrimary ? 'Responsible' : 'Set as responsible'}
                >
                  <Star
                    className={cn(
                      'h-3.5 w-3.5',
                      isPrimary && 'fill-primary text-primary',
                    )}
                  />
                </button>
              </li>
            )
          })}
        </ul>
      ) : (
        <span className="text-sm text-muted-foreground">No assignees</span>
      )}

      <Popover open={pickerOpen} onOpenChange={setPickerOpen}>
        <PopoverTrigger asChild>
          <Button
            variant="outline"
            size="sm"
            className="w-full justify-start"
            disabled={!teamID}
          >
            <UserPlus className="h-3.5 w-3.5" />
            Assignees
          </Button>
        </PopoverTrigger>
        <PopoverContent className="w-64 p-0" align="start">
          <Command>
            <CommandInput placeholder="Search members…" />
            <CommandList>
              <CommandEmpty>
                {teamMembersQuery.isLoading
                  ? 'Loading…'
                  : 'No team members.'}
              </CommandEmpty>
              <CommandGroup>
                {teamMembers.map((tm) => {
                  const userId = tm.user_id ?? ''
                  if (!userId) return null
                  const member = memberMap.get(userId)
                  const isAssigned = assignedIds.has(userId)
                  return (
                    <CommandItem
                      key={userId}
                      value={`${memberName(member, userId)} ${userId}`}
                      onSelect={() => toggleAssignee(userId, isAssigned)}
                      className="gap-2"
                    >
                      <Avatar className="h-5 w-5">
                        <AvatarFallback className="text-[10px]">
                          {memberInitials(member)}
                        </AvatarFallback>
                      </Avatar>
                      <span className="min-w-0 flex-1 truncate">
                        {memberName(member, userId)}
                      </span>
                      {isAssigned ? (
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
