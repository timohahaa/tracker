import { Menu } from 'lucide-react'

import { ThemeToggle } from '@/components/theme-toggle'
import { Avatar, AvatarFallback } from '@/components/ui/avatar'
import { Button } from '@/components/ui/button'
import { useCurrentUser } from '@/features/auth/session-provider'
import { useCurrentWorkspace } from '@/features/auth/workspace-context'

// Derives up-to-two-letter initials from a display name, falling back to the
// email local part, then a generic glyph.
function initials(name: string): string {
  const parts = name.trim().split(/\s+/).filter(Boolean)
  if (parts.length === 0) return '?'
  if (parts.length === 1) return parts[0].slice(0, 2).toUpperCase()
  return (parts[0][0] + parts[parts.length - 1][0]).toUpperCase()
}

interface TopbarProps {
  // Opens the mobile navigation drawer (only rendered on narrow screens).
  onOpenMobileNav: () => void
}

export function Topbar({ onOpenMobileNav }: TopbarProps) {
  const { slug } = useCurrentWorkspace()
  const user = useCurrentUser()

  const displayName = user?.display_name || user?.email || 'Account'
  const email = user?.email ?? ''

  return (
    <header className="flex h-14 shrink-0 items-center gap-3 border-b bg-background px-4">
      <Button
        variant="ghost"
        size="icon"
        className="md:hidden"
        onClick={onOpenMobileNav}
        aria-label="Open navigation menu"
      >
        <Menu className="h-5 w-5" />
      </Button>

      <div className="min-w-0 flex-1">
        <p className="truncate text-sm font-medium">{slug}</p>
      </div>

      <ThemeToggle />

      <div className="flex items-center gap-2">
        <div className="hidden text-right leading-tight sm:block">
          <p className="truncate text-sm font-medium">{displayName}</p>
          {email && (
            <p className="truncate text-xs text-muted-foreground">{email}</p>
          )}
        </div>
        <Avatar className="h-8 w-8">
          <AvatarFallback className="text-xs">
            {initials(displayName)}
          </AvatarFallback>
        </Avatar>
      </div>
    </header>
  )
}
