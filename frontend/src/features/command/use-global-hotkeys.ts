import { useEffect } from 'react'

interface HotkeyHandlers {
  // Cmd/Ctrl+K — works even while typing.
  onTogglePalette: () => void
  onOpenPalette: () => void
  onCreateIssue: () => void
  onOpenHelp: () => void
  // Receives the path segment for a "go to" jump ('' = workspace home).
  onNavigate: (segment: string) => void
  // When false, single-key shortcuts are ignored (e.g. an overlay owns the
  // keyboard). Cmd/Ctrl+K still toggles the palette regardless.
  enabled: boolean
}

// `g <key>` destinations (Linear-style). '' targets the workspace home.
const GO_TARGETS: Record<string, string> = {
  h: '',
  i: 'issues',
  p: 'projects',
  t: 'teams',
  s: 'settings',
}

// True when the event originates from a text-entry surface, so single-key
// shortcuts must not hijack the keystroke.
function isEditableTarget(el: EventTarget | null): boolean {
  if (!(el instanceof HTMLElement)) return false
  const tag = el.tagName
  return (
    tag === 'INPUT' ||
    tag === 'TEXTAREA' ||
    tag === 'SELECT' ||
    el.isContentEditable
  )
}

// Installs the app-wide keyboard shortcuts on `document`. See ShortcutsHelp for
// the user-facing list.
export function useGlobalHotkeys({
  onTogglePalette,
  onOpenPalette,
  onCreateIssue,
  onOpenHelp,
  onNavigate,
  enabled,
}: HotkeyHandlers) {
  useEffect(() => {
    let goMode = false
    let goTimer: ReturnType<typeof setTimeout> | undefined

    const clearGo = () => {
      goMode = false
      if (goTimer) clearTimeout(goTimer)
    }

    function handleKeyDown(event: KeyboardEvent) {
      // Cmd/Ctrl+K toggles the palette from anywhere, including inputs.
      if ((event.metaKey || event.ctrlKey) && event.key.toLowerCase() === 'k') {
        event.preventDefault()
        onTogglePalette()
        return
      }

      // Everything below is a single-key shortcut: skip when modified, disabled,
      // or while typing.
      if (event.metaKey || event.ctrlKey || event.altKey) return
      if (!enabled) return
      if (isEditableTarget(event.target)) return

      if (goMode) {
        const target = GO_TARGETS[event.key.toLowerCase()]
        clearGo()
        if (target !== undefined) {
          event.preventDefault()
          onNavigate(target)
        }
        return
      }

      switch (event.key) {
        case '/':
          event.preventDefault()
          onOpenPalette()
          break
        case 'c':
        case 'C':
          event.preventDefault()
          onCreateIssue()
          break
        case '?':
          event.preventDefault()
          onOpenHelp()
          break
        case 'g':
        case 'G':
          goMode = true
          goTimer = setTimeout(clearGo, 1200)
          break
        default:
          break
      }
    }

    document.addEventListener('keydown', handleKeyDown)
    return () => {
      document.removeEventListener('keydown', handleKeyDown)
      clearGo()
    }
  }, [
    onTogglePalette,
    onOpenPalette,
    onCreateIssue,
    onOpenHelp,
    onNavigate,
    enabled,
  ])
}
