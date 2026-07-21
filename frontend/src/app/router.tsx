import { createBrowserRouter, Outlet } from 'react-router-dom'

// Placeholder root layout. Real app-shell, routes and guards land in Phase 1+.
function RootLayout() {
  return <Outlet />
}

export const router = createBrowserRouter([
  {
    path: '/',
    element: <RootLayout />,
    children: [],
  },
])
