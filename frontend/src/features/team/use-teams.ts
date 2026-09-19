import { useQueryClient } from '@tanstack/react-query'

import { $api } from '@/api/client'
import type { components } from '@/api/generated/schema'

export type Team = components['schemas']['internal_handler_team.teamResponse']
export type CreateTeamBody =
  components['schemas']['internal_handler_team.createTeamRequest']
export type UpdateTeamBody =
  components['schemas']['internal_handler_team.updateTeamRequest']

// Shared query-key prefixes. openapi-react-query keys are `['get', path, params?]`,
// so a `['get', path]` prefix matches every param variant for invalidation.
export const TEAMS_QUERY_KEY = [
  'get',
  '/api/workspaces/{workspaceID}/teams',
] as const

export const TEAM_QUERY_KEY = [
  'get',
  '/api/workspaces/{workspaceID}/teams/{teamID}',
] as const

// GET teams of a workspace. Access: any member.
export function useTeams(workspaceID: string) {
  return $api.useQuery('get', '/api/workspaces/{workspaceID}/teams', {
    params: { path: { workspaceID } },
  })
}

// GET one team. Access: any member.
export function useTeam(workspaceID: string, teamID: string) {
  return $api.useQuery(
    'get',
    '/api/workspaces/{workspaceID}/teams/{teamID}',
    { params: { path: { workspaceID, teamID } } },
    { enabled: !!teamID },
  )
}

// POST create a team. Backend auto-creates the default statuses. Access: member.
// Errors: 400 (key format), 409 (key taken).
export function useCreateTeam() {
  const queryClient = useQueryClient()

  return $api.useMutation('post', '/api/workspaces/{workspaceID}/teams', {
    onSuccess: () =>
      queryClient.invalidateQueries({ queryKey: TEAMS_QUERY_KEY }),
  })
}

// PATCH update a team (name/key/description). Access: team member.
export function useUpdateTeam() {
  const queryClient = useQueryClient()

  return $api.useMutation(
    'patch',
    '/api/workspaces/{workspaceID}/teams/{teamID}',
    {
      onSuccess: () => {
        queryClient.invalidateQueries({ queryKey: TEAMS_QUERY_KEY })
        queryClient.invalidateQueries({ queryKey: TEAM_QUERY_KEY })
      },
    },
  )
}

// DELETE (soft) a team. Access: workspace owner/admin.
export function useDeleteTeam() {
  const queryClient = useQueryClient()

  return $api.useMutation(
    'delete',
    '/api/workspaces/{workspaceID}/teams/{teamID}',
    {
      onSuccess: () =>
        queryClient.invalidateQueries({ queryKey: TEAMS_QUERY_KEY }),
    },
  )
}
