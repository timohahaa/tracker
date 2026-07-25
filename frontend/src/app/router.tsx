import { lazy, Suspense } from 'react'
import {
  createBrowserRouter,
  Navigate,
  Outlet,
  type RouteObject,
} from 'react-router-dom'

import { RouteErrorBoundary } from '@/components/common/route-error-boundary'
import { RouteFallback } from '@/components/common/skeletons'
import { ProtectedRoute } from '@/features/auth/protected-route'

const LoginPage = lazy(() => import('@/pages/login-page'))
const RegisterPage = lazy(() => import('@/pages/register-page'))
const WorkspaceHomePage = lazy(() => import('@/pages/workspace-home-page'))
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
// The real app-shell (sidebar/topbar) arrives in Phase 1.4.
function RootLayout() {
  return <Outlet />
}

export const router = createBrowserRouter([
  {
    path: '/',
    element: <RootLayout />,
    errorElement: <RouteErrorBoundary />,
    children: [
      // Temporary landing redirect. Phase 1.3 replaces this with real workspace
      // resolution (session -> default workspace slug).
      { index: true, element: <Navigate to="/login" replace /> },

      // Public routes (stubs until Phase 2.1).
      { path: 'login', element: lazyElement(LoginPage) },
      { path: 'register', element: lazyElement(RegisterPage) },

      // Protected workspace subtree. ProtectedRoute renders <Outlet /> only when
      // the (placeholder) session is authenticated, otherwise -> /login.
      {
        path: ':workspaceSlug',
        element: <ProtectedRoute />,
        children: [
          { index: true, element: lazyElement(WorkspaceHomePage) },
          { path: '*', element: lazyElement(NotFoundPage) },
        ],
      },

      // Catch-all 404.
      { path: '*', element: lazyElement(NotFoundPage) },
    ],
  },
])
