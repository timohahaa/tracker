import { useState } from 'react'
import { toast } from 'sonner'

import { AppError } from '@/api/client'
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
} from '@/components/ui/dialog'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { useCurrentUser } from '@/features/auth/session-provider'
import { useCurrentWorkspace } from '@/features/auth/workspace-context'
import { AuthErrorBanner } from '@/pages/auth-shell'

import {
  type Member,
  type MemberRole,
  useRemoveMember,
  useUpdateMember,
} from './use-members'

// Two-letter initials from a display name, falling back to the email local part.
function initials(member: Member): string {
  const source = member.display_name?.trim() || member.email || ''
  const parts = source.split(/\s+/).filter(Boolean)
  if (parts.length === 0) return '?'
  if (parts.length === 1) return parts[0].slice(0, 2).toUpperCase()
  return (parts[0][0] + parts[parts.length - 1][0]).toUpperCase()
}

// Roles selectable via the update-member endpoint. The owner role is never
// assignable (backend forbids it), so owners render as a static badge.
const ASSIGNABLE_ROLES: { value: MemberRole; label: string }[] = [
  { value: 'admin', label: 'Admin' },
  { value: 'member', label: 'Member' },
]

function roleLabel(role: string | undefined): string {
  switch (role) {
    case 'owner':
      return 'Owner'
    case 'admin':
      return 'Admin'
    case 'member':
      return 'Member'
    default:
      return role || 'Member'
  }
}

function RoleBadge({ role }: { role: string | undefined }) {
  return (
    <Badge variant={role === 'owner' ? 'default' : 'secondary'}>
      {roleLabel(role)}
    </Badge>
  )
}

interface MemberRowProps {
  member: Member
  workspaceID: string
  // Whether the viewer can manage members (owner/admin).
  canManage: boolean
  // The viewer's own user id, to guard self-actions.
  currentUserId: string | undefined
  // True when this is the workspace's sole owner (cannot be removed/demoted).
  isLastOwner: boolean
}

function MemberRow({
  member,
  workspaceID,
  canManage,
  currentUserId,
  isLastOwner,
}: MemberRowProps) {
  const [confirmOpen, setConfirmOpen] = useState(false)
  const updateMember = useUpdateMember()
  const removeMember = useRemoveMember()

  const userID = member.user_id ?? ''
  const isSelf = !!currentUserId && userID === currentUserId
  const isOwner = member.role === 'owner'

  // Owners can't be re-roled here (owner is not assignable, and demoting the
  // last owner is blocked); you also can't change your own role.
  const canEditRole = canManage && !isOwner && !isSelf
  // Removal is blocked for yourself and for the last owner (backend backstops).
  const canRemove = canManage && !isSelf && !(isOwner && isLastOwner)

  const handleRoleChange = (role: string) => {
    updateMember.mutate(
      { params: { path: { workspaceID, userID } }, body: { role: role as MemberRole } },
      {
        onSuccess: () => toast.success('Member role updated'),
        onError: (error) =>
          toast.error(
            error instanceof AppError
              ? error.message
              : 'Failed to update member',
          ),
      },
    )
  }

  const handleRemove = () => {
    removeMember.mutate(
      { params: { path: { workspaceID, userID } } },
      {
        onSuccess: () => {
          toast.success('Member removed')
          setConfirmOpen(false)
        },
      },
    )
  }

  const removeError =
    removeMember.error instanceof AppError ? removeMember.error : null
  const displayName = member.display_name?.trim() || member.email || 'Member'

  return (
    <div className="flex items-center gap-3 border-b py-3 last:border-b-0">
      <Avatar className="h-9 w-9 shrink-0">
        <AvatarFallback className="text-xs">{initials(member)}</AvatarFallback>
      </Avatar>

      <div className="min-w-0 flex-1">
        <p className="truncate text-sm font-medium">
          {displayName}
          {isSelf ? (
            <span className="ml-1.5 text-xs text-muted-foreground">(you)</span>
          ) : null}
        </p>
        <p className="truncate text-xs text-muted-foreground">
          {member.email}
          {member.title ? ` · ${member.title}` : ''}
        </p>
      </div>

      <div className="flex shrink-0 items-center gap-2">
        {canEditRole ? (
          <Select
            value={member.role}
            onValueChange={handleRoleChange}
            disabled={updateMember.isPending}
          >
            <SelectTrigger className="h-8 w-28" aria-label="Member role">
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              {ASSIGNABLE_ROLES.map((r) => (
                <SelectItem key={r.value} value={r.value}>
                  {r.label}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
        ) : (
          <RoleBadge role={member.role} />
        )}

        {canRemove ? (
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
                <DialogTitle>Remove member</DialogTitle>
                <DialogDescription>
                  Remove <span className="font-medium">{displayName}</span> from
                  this workspace? They will lose access immediately.
                </DialogDescription>
              </DialogHeader>
              {removeError ? (
                <AuthErrorBanner message={removeError.message} />
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

interface MembersListProps {
  members: Member[]
}

export function MembersList({ members }: MembersListProps) {
  const { id: workspaceID, role } = useCurrentWorkspace()
  const currentUserId = useCurrentUser()?.id

  const canManage = role === 'owner' || role === 'admin'
  const ownerCount = members.filter((m) => m.role === 'owner').length

  return (
    <div className="rounded-lg border px-4">
      {members.map((member) => (
        <MemberRow
          key={member.user_id}
          member={member}
          workspaceID={workspaceID}
          canManage={canManage}
          currentUserId={currentUserId}
          isLastOwner={ownerCount <= 1}
        />
      ))}
    </div>
  )
}
