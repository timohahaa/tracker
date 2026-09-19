import { lazy, Suspense } from 'react'
import {
  createBrowserRouter,
  Outlet,
  type RouteObject,
} from 'react-router-dom'

import { RouteErrorBoundary } from '@/components/common/route-error-boundary'
import { RouteFallback } from '@/components/common/skeletons'
import { ProtectedRoute } from '@/features/auth/protected-route'
import { RootIndexRedirect } from '@/features/auth/root-index-redirect'
import { SessionProvider } from '@/features/auth/session-provider'

const LoginPage = lazy(() => import('@/pages/login-page'))
const RegisterPage = lazy(() => import('@/pages/register-page'))
const WorkspaceHomePage = lazy(() => import('@/pages/workspace-home-page'))
const SettingsLayout = lazy(() => import('@/pages/settings-layout'))
const WorkspaceSettingsPage = lazy(
  () => import('@/pages/workspace-settings-page'),
)
const WorkspaceMembersPage = lazy(
  () => import('@/pages/workspace-members-page'),
)
const TeamsPage = lazy(() => import('@/pages/teams-page'))
const TeamDetailPage = lazy(() => import('@/pages/team-detail-page'))
const IssuesPage = lazy(() => import('@/pages/issues-page'))
const IssueDetailPage = lazy(() => import('@/pages/issue-detail-page'))
const NotFoundPage = lazy(() => import('@/pages/not-found-page'))

// Wraps a lazy page element in the shared Suspense fallback so route-level
// code splitting always has a loading state.
function lazyElement(Element: React.ComponentType): RouteObject['element'] {
  return (
    <Suspense fallback={<RouteFallback />}>
      <Element />
    </Suspense>
  )
}

// Root layout exists so the errorElement below covers the whole route tree.
// SessionProvider is mounted here so it sits inside both the QueryClientProvider
// (server-state hooks) and the RouterProvider (router hooks) while wrapping every
// route. The real app-shell (sidebar/topbar) arrives in Phase 1.4.
function RootLayout() {
  return (
    <SessionProvider>
      <Outlet />
    </SessionProvider>
  )
}

export const router = createBrowserRouter([
  {
    path: '/',
    element: <RootLayout />,
    errorElement: <RouteErrorBoundary />,
    children: [
      // Landing redirect: authenticated -> default workspace slug, otherwise
      // -> /login (resolved from the session bootstrap).
      { index: true, element: <RootIndexRedirect /> },

      // Public routes (stubs until Phase 2.1).
      { path: 'login', element: lazyElement(LoginPage) },
      { path: 'register', element: lazyElement(RegisterPage) },

      // Protected workspace subtree. ProtectedRoute renders <Outlet /> only when
      // the session is authenticated and the slug resolves, otherwise redirects.
      {
        path: ':workspaceSlug',
        element: <ProtectedRoute />,
        children: [
          { index: true, element: lazyElement(WorkspaceHomePage) },
          // Issues (2.4.1): workspace-wide list with filters/sort/search + cursor
          // infinite scroll. Detail/create/edit (2.4.2) live on the child route,
          // keyed by identifier (e.g. ENG-123) or UUID.
          { path: 'issues', element: lazyElement(IssuesPage) },
          { path: 'issues/:identifier', element: lazyElement(IssueDetailPage) },
          // Teams (2.3): list + detail (keyed by team UUID). The detail page
          // owns its own Settings/Statuses/Members tabs.
          { path: 'teams', element: lazyElement(TeamsPage) },
          { path: 'teams/:teamId', element: lazyElement(TeamDetailPage) },
          // Workspace Settings area (2.2). A layout route with sub-section tabs
          // (General/Members); 2.7 adds GitLab + tokens as sibling children.
          {
            path: 'settings',
            element: lazyElement(SettingsLayout),
            children: [
              { index: true, element: lazyElement(WorkspaceSettingsPage) },
              { path: 'members', element: lazyElement(WorkspaceMembersPage) },
            ],
          },
          { path: '*', element: lazyElement(NotFoundPage) },
        ],
      },

      // Catch-all 404.
      { path: '*', element: lazyElement(NotFoundPage) },
    ],
  },
])
