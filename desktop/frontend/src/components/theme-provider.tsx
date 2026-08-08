import { createContext, useContext, useEffect, useState } from 'react'

export type Theme = 'dark' | 'light' | 'system'

type ThemeProviderState = {
  theme: Theme
  resolvedTheme: 'dark' | 'light'
  setTheme: (theme: Theme) => void
}

const ThemeProviderContext = createContext<ThemeProviderState | undefined>(undefined)

const STORAGE_KEY = 'agq-theme'
const LEGACY_STORAGE_KEY = 'antigravity-token-monitor-theme'

function systemTheme(): 'dark' | 'light' {
  return window.matchMedia('(prefers-color-scheme: dark)').matches ? 'dark' : 'light'
}

export function ThemeProvider({
  children,
  defaultTheme = 'dark',
}: {
  children: React.ReactNode
  defaultTheme?: Theme
}) {
  const [theme, setThemeState] = useState<Theme>(
    () =>
      (localStorage.getItem(STORAGE_KEY) as Theme) ||
      (localStorage.getItem(LEGACY_STORAGE_KEY) as Theme) ||
      defaultTheme,
  )
  const resolvedTheme = theme === 'system' ? systemTheme() : theme

  useEffect(() => {
    const root = window.document.documentElement
    const apply = () => {
      const resolved = theme === 'system' ? systemTheme() : theme
      root.classList.toggle('dark', resolved === 'dark')
    }
    apply()
    if (theme === 'system') {
      const mql = window.matchMedia('(prefers-color-scheme: dark)')
      mql.addEventListener('change', apply)
      return () => mql.removeEventListener('change', apply)
    }
  }, [theme])

  const setTheme = (next: Theme) => {
    localStorage.setItem(STORAGE_KEY, next)
    localStorage.removeItem(LEGACY_STORAGE_KEY)
    setThemeState(next)
  }

  return (
    <ThemeProviderContext.Provider value={{ theme, resolvedTheme, setTheme }}>
      {children}
    </ThemeProviderContext.Provider>
  )
}

export function useTheme() {
  const context = useContext(ThemeProviderContext)
  if (!context) throw new Error('useTheme must be used within a ThemeProvider')
  return context
}
