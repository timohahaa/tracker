import { useQueryClient } from '@tanstack/react-query'

import { $api } from '@/api/client'
import type { components } from '@/api/generated/schema'

// Team member rows carry only `{ user_id, role, team_id, created_at }` — no
// name/email. Callers join against workspace members (2.2) for display info.
export type TeamMember =
  components['schemas']['internal_handler_team.memberResponse']
export type AddTeamMemberBody =
  components['schemas']['internal_handler_team.addMemberRequest']
export type TeamRole = NonNullable<AddTeamMemberBody['role']>

export const TEAM_MEMBERS_QUERY_KEY = [
  'get',
  '/api/workspaces/{workspaceID}/teams/{teamID}/members',
] as const

// GET team members. Access: any workspace member.
export function useTeamMembers(workspaceID: string, teamID: string) {
  return $api.useQuery(
    'get',
    '/api/workspaces/{workspaceID}/teams/{teamID}/members',
    { params: { path: { workspaceID, teamID } } },
    { enabled: !!teamID },
  )
}

// POST add a team member (from the workspace member pool). Access: team member.
export function useAddTeamMember() {
  const queryClient = useQueryClient()

  return $api.useMutation(
    'post',
    '/api/workspaces/{workspaceID}/teams/{teamID}/members',
    {
      onSuccess: () =>
        queryClient.invalidateQueries({ queryKey: TEAM_MEMBERS_QUERY_KEY }),
    },
  )
}

// DELETE (soft) a team member. Access: team member.
export function useRemoveTeamMember() {
  const queryClient = useQueryClient()

  return $api.useMutation(
    'delete',
    '/api/workspaces/{workspaceID}/teams/{teamID}/members/{userID}',
    {
      onSuccess: () =>
        queryClient.invalidateQueries({ queryKey: TEAM_MEMBERS_QUERY_KEY }),
    },
  )
}
