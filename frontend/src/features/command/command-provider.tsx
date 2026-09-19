import {
  createContext,
  useCallback,
  useContext,
  useMemo,
  useState,
} from 'react'
import { useNavigate } from 'react-router-dom'

import { useCurrentWorkspace } from '@/features/auth/workspace-context'
import { CreateIssueDialog } from '@/features/issue/create-issue-dialog'

import { CommandPalette } from './command-palette'
import { ShortcutsHelp } from './shortcuts-help'
import { useGlobalHotkeys } from './use-global-hotkeys'

interface CommandContextValue {
  openPalette: () => void
  openCreateIssue: () => void
  openHelp: () => void
}

const CommandContext = createContext<CommandContextValue | null>(null)

// Access the command actions (palette, new-issue, help) from anywhere in the
// app shell — e.g. the topbar's search button.
export function useCommands(): CommandContextValue {
  const ctx = useContext(CommandContext)
  if (!ctx) {
    throw new Error('useCommands must be used within a CommandProvider')
  }
  return ctx
}

// Owns the app-wide command surfaces: the Cmd/Ctrl+K palette, the (reused)
// "New issue" dialog, and the keyboard-shortcut help overlay, plus the global
// hotkeys that open them. Mounted inside the app-shell so it has workspace
// context and router access. Only one overlay is open at a time.
export function CommandProvider({ children }: { children: React.ReactNode }) {
  const { slug } = useCurrentWorkspace()
  const navigate = useNavigate()

  const [paletteOpen, setPaletteOpen] = useState(false)
  const [createIssueOpen, setCreateIssueOpen] = useState(false)
  const [helpOpen, setHelpOpen] = useState(false)

  const openPalette = useCallback(() => {
    setHelpOpen(false)
    setCreateIssueOpen(false)
    setPaletteOpen(true)
  }, [])

  const togglePalette = useCallback(() => {
    setPaletteOpen((prev) => !prev)
  }, [])

  const openCreateIssue = useCallback(() => {
    setPaletteOpen(false)
    setHelpOpen(false)
    setCreateIssueOpen(true)
  }, [])

  const openHelp = useCallback(() => {
    setPaletteOpen(false)
    setCreateIssueOpen(false)
    setHelpOpen(true)
  }, [])

  const navigateTo = useCallback(
    (segment: string) => {
      navigate(segment ? `/${slug}/${segment}` : `/${slug}`)
    },
    [navigate, slug],
  )

  const anyOverlayOpen = paletteOpen || createIssueOpen || helpOpen

  useGlobalHotkeys({
    onTogglePalette: togglePalette,
    onOpenPalette: openPalette,
    onCreateIssue: openCreateIssue,
    onOpenHelp: openHelp,
    onNavigate: navigateTo,
    enabled: !anyOverlayOpen,
  })

  const value = useMemo(
    () => ({ openPalette, openCreateIssue, openHelp }),
    [openPalette, openCreateIssue, openHelp],
  )

  return (
    <CommandContext.Provider value={value}>
      {children}
      {/* Palette and create-issue dialog are mounted only while open so their
          data hooks stay idle until invoked. */}
      {paletteOpen ? (
        <CommandPalette
          open={paletteOpen}
          onOpenChange={setPaletteOpen}
          onCreateIssue={openCreateIssue}
        />
      ) : null}
      {createIssueOpen ? (
        <CreateIssueDialog
          open={createIssueOpen}
          onOpenChange={setCreateIssueOpen}
          showTrigger={false}
        />
      ) : null}
      <ShortcutsHelp open={helpOpen} onOpenChange={setHelpOpen} />
    </CommandContext.Provider>
  )
}
