import { createContext } from 'react'

export type Theme = 'light' | 'dark' | 'system'
export type ResolvedTheme = 'light' | 'dark'

export interface ThemeContextValue {
  /** User selection: explicit light/dark or follow the OS ('system'). */
  theme: Theme
  /** Effective theme actually applied to <html> (never 'system'). */
  resolvedTheme: ResolvedTheme
  setTheme: (theme: Theme) => void
  /** Flip between light and dark based on the current resolved theme. */
  toggleTheme: () => void
}

/** Keep in sync with the anti-FOUC inline script in index.html. */
export const THEME_STORAGE_KEY = 'ui-theme'

export const ThemeContext = createContext<ThemeContextValue | undefined>(undefined)
