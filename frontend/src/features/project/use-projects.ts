import { useQueryClient } from '@tanstack/react-query'

import { $api } from '@/api/client'
import type { components } from '@/api/generated/schema'

import { PROJECTS_QUERY_KEY } from './use-projects-list'

export type CreateProjectBody =
  components['schemas']['internal_handler_project.createProjectRequest']
export type UpdateProjectBody =
  components['schemas']['internal_handler_project.updateProjectRequest']
// Project status enum (planned | in_progress | completed | canceled). Narrower
// than the request DTO's optional field.
export type ProjectStatus = NonNullable<CreateProjectBody['status']>

// Shared query-key prefix for the single-project endpoint. openapi-react-query
// keys are `['get', path, init]`, so this `['get', path]` prefix matches every
// projectID variant for invalidation. Distinct from PROJECTS_QUERY_KEY (the list
// prefix), which does not overlap because the path segment differs.
export const PROJECT_QUERY_KEY = [
  'get',
  '/api/workspaces/{workspaceID}/projects/{projectID}',
] as const

// Exact query key for one project, matching what `useProject` builds. Needed for
// `setQueryData` where a prefix won't do.
export function projectQueryKey(workspaceID: string, projectID: string) {
  return [
    'get',
    '/api/workspaces/{workspaceID}/projects/{projectID}',
    { params: { path: { workspaceID, projectID } } },
  ] as const
}

const PAGE_SIZE = 30

// Cursor-paginated projects list for the projects page. First page sends no
// cursor: `initialPageParam` is `null` (not `undefined`, which openapi-react-query
// coerces to `0` and the backend 400s), and openapi-fetch omits null query
// params. Subsequent pages use `pagination.next_cursor` while `has_more` is true.
// Access: any workspace member. The thin single-page `useProjectsList` (pickers)
// stays; both share PROJECTS_QUERY_KEY for invalidation.
export function useProjects(workspaceID: string) {
  return $api.useInfiniteQuery(
    'get',
    '/api/workspaces/{workspaceID}/projects',
    { params: { path: { workspaceID }, query: { limit: PAGE_SIZE } } },
    {
      getNextPageParam: (lastPage) =>
        lastPage.pagination?.has_more && lastPage.pagination.next_cursor
          ? lastPage.pagination.next_cursor
          : undefined,
      initialPageParam: null,
      pageParamName: 'cursor',
    },
  )
}

// GET one project by UUID. Access: any workspace member.
export function useProject(workspaceID: string, projectID: string) {
  return $api.useQuery(
    'get',
    '/api/workspaces/{workspaceID}/projects/{projectID}',
    { params: { path: { workspaceID, projectID } } },
    { enabled: !!workspaceID && !!projectID },
  )
}

// POST create a project. Access: any workspace member. Errors: 400 validation.
export function useCreateProject() {
  const queryClient = useQueryClient()

  return $api.useMutation('post', '/api/workspaces/{workspaceID}/projects', {
    onSuccess: () =>
      queryClient.invalidateQueries({ queryKey: PROJECTS_QUERY_KEY }),
  })
}

// PATCH update a project (partial; `team_ids` uses replace-semantics). Writes the
// returned project into the single-project cache, then invalidates the list.
// Access: any workspace member. Errors: 400/404.
export function useUpdateProject() {
  const queryClient = useQueryClient()

  return $api.useMutation(
    'patch',
    '/api/workspaces/{workspaceID}/projects/{projectID}',
    {
      onSuccess: (data, variables) => {
        const { workspaceID, projectID } = variables.params.path
        queryClient.setQueryData(projectQueryKey(workspaceID, projectID), data)
        queryClient.invalidateQueries({ queryKey: PROJECTS_QUERY_KEY })
      },
    },
  )
}

// DELETE (soft) a project. Access: workspace owner/admin. Errors: 403/404.
export function useDeleteProject() {
  const queryClient = useQueryClient()

  return $api.useMutation(
    'delete',
    '/api/workspaces/{workspaceID}/projects/{projectID}',
    {
      onSuccess: (_data, variables) => {
        const { workspaceID, projectID } = variables.params.path
        queryClient.removeQueries({
          queryKey: projectQueryKey(workspaceID, projectID),
        })
        queryClient.invalidateQueries({ queryKey: PROJECTS_QUERY_KEY })
      },
    },
  )
}

// DELETE the project lead (lead_id -> null). Idempotent, returns the updated
// project. PATCH can't clear a nullable FK (null/omitted means "don't touch"), so
// clearing the lead goes through here. Access: any workspace member.
export function useClearProjectLead() {
  const queryClient = useQueryClient()

  return $api.useMutation(
    'delete',
    '/api/workspaces/{workspaceID}/projects/{projectID}/lead',
    {
      onSuccess: (data, variables) => {
        const { workspaceID, projectID } = variables.params.path
        queryClient.setQueryData(projectQueryKey(workspaceID, projectID), data)
        queryClient.invalidateQueries({ queryKey: PROJECTS_QUERY_KEY })
      },
    },
  )
}
