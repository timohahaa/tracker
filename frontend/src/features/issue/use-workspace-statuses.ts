import { useQueries } from '@tanstack/react-query'
import { useMemo } from 'react'

import { $api } from '@/api/client'
import { type StatusGroup } from '@/lib/status'

export interface StatusInfo {
  group: StatusGroup
  name: string
}

// Issues carry `status_id` but not `status_group`, and statuses are per-team, so
// grouping the workspace-wide issue list needs every team's statuses. Fetch them
// in parallel (one cached query per team, shared with the team-detail views) and
// fold into a `status_id -> { group, name }` lookup. Access: any member.
export function useWorkspaceStatuses(workspaceID: string, teamIDs: string[]) {
  const results = useQueries({
    queries: teamIDs.map((teamID) =>
      $api.queryOptions(
        'get',
        '/api/workspaces/{workspaceID}/teams/{teamID}/statuses',
        { params: { path: { workspaceID, teamID } } },
      ),
    ),
  })

  const isLoading = results.some((r) => r.isLoading)

  const statusMap = useMemo(() => {
    const map = new Map<string, StatusInfo>()
    for (const result of results) {
      for (const status of result.data?.data ?? []) {
        if (status.id) {
          map.set(status.id, {
            group: (status.group as StatusGroup) ?? 'backlog',
            name: status.name ?? '',
          })
        }
      }
    }
    return map
    // `results` is a fresh array each render; the map build is cheap.
     
  }, [results])

  return { statusMap, isLoading }
}
