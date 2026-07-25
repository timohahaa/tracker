import { useEffect, useState } from 'react'
import { Outlet, useLocation } from 'react-router-dom'

import {
  Sheet,
  SheetContent,
  SheetTitle,
} from '@/components/ui/sheet'

import { SidebarNav } from './sidebar-nav'
import { Topbar } from './topbar'
import { WorkspaceSwitcher } from './workspace-switcher'

// Shared sidebar body: workspace switcher on top, primary nav below. Reused by
// both the fixed desktop rail and the mobile drawer.
function SidebarContent({ onNavigate }: { onNavigate?: () => void }) {
  return (
    <div className="flex h-full flex-col gap-4">
      <WorkspaceSwitcher onSelect={onNavigate} />
      <SidebarNav onNavigate={onNavigate} />
    </div>
  )
}

// App-shell (Phase 1.4). Fixed sidebar + topbar + scrollable content region
// hosting the routed <Outlet/>. On narrow screens the sidebar collapses into a
// slide-in drawer opened from the topbar. Rendered by ProtectedRoute inside the
// WorkspaceProvider, so all children have the current-workspace context.
export function AppShell() {
  const [mobileNavOpen, setMobileNavOpen] = useState(false)
  const location = useLocation()

  // Close the drawer whenever the route changes (e.g. after tapping a link or
  // switching workspace).
  useEffect(() => {
    setMobileNavOpen(false)
  }, [location.pathname])

  return (
    <div className="flex h-svh w-full overflow-hidden">
      <aside className="hidden w-60 shrink-0 flex-col border-r bg-card p-3 md:flex">
        <SidebarContent />
      </aside>

      <Sheet open={mobileNavOpen} onOpenChange={setMobileNavOpen}>
        <SheetContent side="left" className="w-72 p-3">
          <SheetTitle className="sr-only">Navigation</SheetTitle>
          <SidebarContent onNavigate={() => setMobileNavOpen(false)} />
        </SheetContent>
      </Sheet>

      <div className="flex min-w-0 flex-1 flex-col">
        <Topbar onOpenMobileNav={() => setMobileNavOpen(true)} />
        <main className="flex-1 overflow-y-auto">
          <Outlet />
        </main>
      </div>
    </div>
  )
}
