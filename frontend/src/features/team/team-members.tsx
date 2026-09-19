import { UserPlus } from 'lucide-react'
import { useMemo, useState } from 'react'
import { toast } from 'sonner'

import { AppError } from '@/api/client'
import { EmptyState } from '@/components/common/empty-state'
import { ErrorState } from '@/components/common/error-state'
import { ListSkeleton } from '@/components/common/skeletons'
import { Avatar, AvatarFallback } from '@/components/ui/avatar'
import { Badge } from '@/components/ui/badge'
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
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { useCurrentWorkspace } from '@/features/auth/workspace-context'
import { type Member, useMembers } from '@/features/workspace/use-members'

import {
  type TeamMember,
  type TeamRole,
  useAddTeamMember,
  useRemoveTeamMember,
  useTeamMembers,
} from './use-team-members'

function initials(name: string, email: string): string {
  const source = name.trim() || email || ''
  const parts = source.split(/\s+/).filter(Boolean)
  if (parts.length === 0) return '?'
  if (parts.length === 1) return parts[0].slice(0, 2).toUpperCase()
  return (parts[0][0] + parts[parts.length - 1][0]).toUpperCase()
}

function roleLabel(role: string | undefined): string {
  return role === 'owner' ? 'Owner' : 'Member'
}

const TEAM_ROLES: { value: TeamRole; label: string }[] = [
  { value: 'member', label: 'Member' },
  { value: 'owner', label: 'Owner' },
]

interface Joined {
  teamMember: TeamMember
  display: Member | undefined
}

function TeamMemberRow({
  joined,
  teamID,
  canManage,
}: {
  joined: Joined
  teamID: string
  canManage: boolean
}) {
  const { id: workspaceID } = useCurrentWorkspace()
  const removeMember = useRemoveTeamMember()
  const [confirmOpen, setConfirmOpen] = useState(false)

  const userID = joined.teamMember.user_id ?? ''
  const displayName =
    joined.display?.display_name?.trim() ||
    joined.display?.email ||
    'Unknown member'
  const email = joined.display?.email ?? ''

  const error = removeMember.error instanceof AppError ? removeMember.error : null

  const handleRemove = () => {
    removeMember.mutate(
      { params: { path: { workspaceID, teamID, userID } } },
      {
        onSuccess: () => {
          toast.success('Member removed from team')
          setConfirmOpen(false)
        },
      },
    )
  }

  return (
    <div className="flex items-center gap-3 border-b py-3 last:border-b-0">
      <Avatar className="h-9 w-9 shrink-0">
        <AvatarFallback className="text-xs">
          {initials(joined.display?.display_name ?? '', email)}
        </AvatarFallback>
      </Avatar>
      <div className="min-w-0 flex-1">
        <p className="truncate text-sm font-medium">{displayName}</p>
        <p className="truncate text-xs text-muted-foreground">{email}</p>
      </div>
      <div className="flex shrink-0 items-center gap-2">
        <Badge
          variant={joined.teamMember.role === 'owner' ? 'default' : 'secondary'}
        >
          {roleLabel(joined.teamMember.role)}
        </Badge>
        {canManage ? (
          <Dialog open={confirmOpen} onOpenChange={setConfirmOpen}>
            <Button
              variant="ghost"
              size="sm"
              className="text-muted-foreground hover:text-destructive"
              onClick={() => setConfirmOpen(true)}
            >
              Remove
            </Button>
            <DialogContent>
              <DialogHeader>
                <DialogTitle>Remove team member</DialogTitle>
                <DialogDescription>
                  Remove <span className="font-medium">{displayName}</span> from
                  this team?
                </DialogDescription>
              </DialogHeader>
              {error ? (
                <p className="text-sm text-destructive">{error.message}</p>
              ) : null}
              <DialogFooter>
                <Button
                  variant="outline"
                  onClick={() => setConfirmOpen(false)}
                  disabled={removeMember.isPending}
                >
                  Cancel
                </Button>
                <Button
                  variant="destructive"
                  onClick={handleRemove}
                  disabled={removeMember.isPending}
                >
                  {removeMember.isPending ? 'Removing…' : 'Remove'}
                </Button>
              </DialogFooter>
            </DialogContent>
          </Dialog>
        ) : null}
      </div>
    </div>
  )
}

function AddTeamMemberDialog({
  teamID,
  eligible,
}: {
  teamID: string
  // Workspace members not already on the team.
  eligible: Member[]
}) {
  const { id: workspaceID } = useCurrentWorkspace()
  const addMember = useAddTeamMember()
  const [open, setOpen] = useState(false)
  const [userId, setUserId] = useState('')
  const [role, setRole] = useState<TeamRole>('member')

  const handleOpenChange = (next: boolean) => {
    setOpen(next)
    if (next) {
      setUserId('')
      setRole('member')
      addMember.reset()
    }
  }

  const error = addMember.error instanceof AppError ? addMember.error : null

  const handleAdd = () => {
    if (!userId) return
    addMember.mutate(
      {
        params: { path: { workspaceID, teamID } },
        body: { user_id: userId, role },
      },
      {
        onSuccess: () => {
          toast.success('Member added to team')
          handleOpenChange(false)
        },
      },
    )
  }

  return (
    <Dialog open={open} onOpenChange={handleOpenChange}>
      <DialogTrigger asChild>
        <Button size="sm" disabled={eligible.length === 0}>
          <UserPlus className="mr-1.5 h-4 w-4" />
          Add member
        </Button>
      </DialogTrigger>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>Add team member</DialogTitle>
          <DialogDescription>
            Add an existing workspace member to this team.
          </DialogDescription>
        </DialogHeader>
        <div className="space-y-4">
          {error ? (
            <p className="text-sm text-destructive">{error.message}</p>
          ) : null}
          <div className="space-y-1.5">
            <label className="text-sm font-medium">Member</label>
            <Select value={userId} onValueChange={setUserId}>
              <SelectTrigger aria-label="Member">
                <SelectValue placeholder="Select a member" />
              </SelectTrigger>
              <SelectContent>
                {eligible.map((m) => (
                  <SelectItem key={m.user_id} value={m.user_id ?? ''}>
                    {m.display_name?.trim() || m.email || m.user_id}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          </div>
          <div className="space-y-1.5">
            <label className="text-sm font-medium">Role</label>
            <Select value={role} onValueChange={(v) => setRole(v as TeamRole)}>
              <SelectTrigger aria-label="Team role">
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                {TEAM_ROLES.map((r) => (
                  <SelectItem key={r.value} value={r.value}>
                    {r.label}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          </div>
        </div>
        <DialogFooter>
          <Button
            variant="outline"
            onClick={() => handleOpenChange(false)}
            disabled={addMember.isPending}
          >
            Cancel
          </Button>
          <Button onClick={handleAdd} disabled={addMember.isPending || !userId}>
            {addMember.isPending ? 'Adding…' : 'Add member'}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}

interface TeamMembersProps {
  teamID: string
  canManage: boolean
}

export function TeamMembers({ teamID, canManage }: TeamMembersProps) {
  const { id: workspaceID } = useCurrentWorkspace()
  const teamMembersQuery = useTeamMembers(workspaceID, teamID)
  const workspaceMembersQuery = useMembers(workspaceID)

  const teamMembers = useMemo(
    () => teamMembersQuery.data?.data ?? [],
    [teamMembersQuery.data],
  )
  const workspaceMembers = useMemo(
    () => workspaceMembersQuery.data?.data ?? [],
    [workspaceMembersQuery.data],
  )

  const displayById = useMemo(() => {
    const map = new Map<string, Member>()
    for (const m of workspaceMembers) if (m.user_id) map.set(m.user_id, m)
    return map
  }, [workspaceMembers])

  const joined: Joined[] = useMemo(
    () =>
      teamMembers.map((tm) => ({
        teamMember: tm,
        display: tm.user_id ? displayById.get(tm.user_id) : undefined,
      })),
    [teamMembers, displayById],
  )

  const eligible = useMemo(() => {
    const onTeam = new Set(teamMembers.map((tm) => tm.user_id))
    return workspaceMembers.filter((m) => !onTeam.has(m.user_id))
  }, [teamMembers, workspaceMembers])

  if (teamMembersQuery.isLoading) {
    return <ListSkeleton rows={3} />
  }

  if (teamMembersQuery.isError) {
    return (
      <ErrorState
        error={teamMembersQuery.error}
        onRetry={() => teamMembersQuery.refetch()}
      />
    )
  }

  return (
    <div className="space-y-3">
      <div className="flex items-center justify-between gap-4">
        <p className="text-sm text-muted-foreground">
          {joined.length} {joined.length === 1 ? 'member' : 'members'}
        </p>
        {canManage ? (
          <AddTeamMemberDialog teamID={teamID} eligible={eligible} />
        ) : null}
      </div>

      {joined.length === 0 ? (
        <EmptyState
          icon={UserPlus}
          title="No team members yet"
          description="Add workspace members to this team to assign them issues."
        />
      ) : (
        <div className="rounded-lg border px-4">
          {joined.map((j) => (
            <TeamMemberRow
              key={j.teamMember.user_id}
              joined={j}
              teamID={teamID}
              canManage={canManage}
            />
          ))}
        </div>
      )}
    </div>
  )
}
