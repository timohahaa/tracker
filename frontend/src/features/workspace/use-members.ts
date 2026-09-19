import { useQueryClient } from '@tanstack/react-query'

import { $api } from '@/api/client'
import type { components } from '@/api/generated/schema'

export type Member =
  components['schemas']['internal_handler_workspace.memberResponse']
export type UpdateMemberBody =
  components['schemas']['internal_handler_workspace.updateMemberRequest']
export type MemberRole = NonNullable<UpdateMemberBody['role']>

// Shared query-key prefix for the members list. Used both to read and to
// invalidate after member mutations (prefix match covers the params variant).
export const MEMBERS_QUERY_KEY = [
  'get',
  '/api/workspaces/{workspaceID}/members',
] as const

// GET workspace members. Access: any member.
export function useMembers(workspaceID: string) {
  return $api.useQuery('get', '/api/workspaces/{workspaceID}/members', {
    params: { path: { workspaceID } },
  })
}

// PATCH a member (role/username/title). Access: owner/admin. Backend rejects
// assigning the owner role and demoting/removing the last owner (403).
export function useUpdateMember() {
  const queryClient = useQueryClient()

  return $api.useMutation(
    'patch',
    '/api/workspaces/{workspaceID}/members/{userID}',
    {
      onSuccess: () =>
        queryClient.invalidateQueries({ queryKey: MEMBERS_QUERY_KEY }),
    },
  )
}

// DELETE (soft) a member. Access: owner/admin. Backend rejects removing the
// last owner (403).
export function useRemoveMember() {
  const queryClient = useQueryClient()

  return $api.useMutation(
    'delete',
    '/api/workspaces/{workspaceID}/members/{userID}',
    {
      onSuccess: () =>
        queryClient.invalidateQueries({ queryKey: MEMBERS_QUERY_KEY }),
    },
  )
}
