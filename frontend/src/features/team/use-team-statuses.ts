import { useQueryClient } from '@tanstack/react-query'

import { $api } from '@/api/client'
import type { components } from '@/api/generated/schema'

export type TeamStatus =
  components['schemas']['internal_handler_team.statusResponse']
export type CreateStatusBody =
  components['schemas']['internal_handler_team.createStatusRequest']
export type UpdateStatusBody =
  components['schemas']['internal_handler_team.updateStatusRequest']

export const TEAM_STATUSES_QUERY_KEY = [
  'get',
  '/api/workspaces/{workspaceID}/teams/{teamID}/statuses',
] as const

// GET team statuses. Access: any workspace member.
export function useTeamStatuses(workspaceID: string, teamID: string) {
  return $api.useQuery(
    'get',
    '/api/workspaces/{workspaceID}/teams/{teamID}/statuses',
    { params: { path: { workspaceID, teamID } } },
    { enabled: !!teamID },
  )
}

// POST create a status in a group. Access: team member.
export function useCreateStatus() {
  const queryClient = useQueryClient()

  return $api.useMutation(
    'post',
    '/api/workspaces/{workspaceID}/teams/{teamID}/statuses',
    {
      onSuccess: () =>
        queryClient.invalidateQueries({ queryKey: TEAM_STATUSES_QUERY_KEY }),
    },
  )
}

// PATCH a status (name/position). Access: team member.
export function useUpdateStatus() {
  const queryClient = useQueryClient()

  return $api.useMutation(
    'patch',
    '/api/workspaces/{workspaceID}/teams/{teamID}/statuses/{statusID}',
    {
      onSuccess: () =>
        queryClient.invalidateQueries({ queryKey: TEAM_STATUSES_QUERY_KEY }),
    },
  )
}

// DELETE a status. Access: team member. Backend rejects removing the last status
// in a group (400) — callers surface a friendly message.
export function useDeleteStatus() {
  const queryClient = useQueryClient()

  return $api.useMutation(
    'delete',
    '/api/workspaces/{workspaceID}/teams/{teamID}/statuses/{statusID}',
    {
      onSuccess: () =>
        queryClient.invalidateQueries({ queryKey: TEAM_STATUSES_QUERY_KEY }),
    },
  )
}
