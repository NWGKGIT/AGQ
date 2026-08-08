import { BarChart3, CircleGauge, Eye, EyeOff, LayoutDashboard, Moon, Settings, Sun } from 'lucide-react'

import { useTheme } from '@/components/theme-provider'
import { Button } from '@/components/ui/button'
import { useAppConfig, useSetAppConfig } from '@/lib/api'
import { cn } from '@/lib/utils'
import type { Page } from '@/types/navigation'

const navItems: { page: Page; label: string; icon: typeof LayoutDashboard }[] = [
  { page: 'overview', label: 'Overview', icon: LayoutDashboard },
  { page: 'analytics', label: 'Analytics', icon: BarChart3 },
  { page: 'settings', label: 'Settings', icon: Settings },
]

// Below the md breakpoint the sidebar collapses to an icons-only rail so the
// app stays usable in narrow tiles on tiling window managers.
export function Sidebar({
  page,
  onNavigate,
}: {
  page: Page
  onNavigate: (page: Page) => void
}) {
  const { resolvedTheme, setTheme } = useTheme()
  const { data: cfg } = useAppConfig()
  const setConfig = useSetAppConfig()
  const masked = cfg?.mask_emails ?? false

  return (
    <aside className="flex h-full w-14 shrink-0 flex-col border-r bg-card md:w-60">
      <div className="flex h-16 items-center justify-center gap-2.5 border-b px-2 md:justify-start md:px-4">
        <div className="flex size-8 shrink-0 items-center justify-center rounded-lg bg-primary text-primary-foreground">
          <CircleGauge className="size-4" aria-hidden="true" />
        </div>
        <div className="hidden min-w-0 leading-tight md:block">
          <span className="block truncate text-sm font-semibold">AGQ</span>
          <span className="block truncate text-xs text-muted-foreground">
            Quota monitor for Antigravity
          </span>
        </div>
      </div>
      <nav className="flex flex-1 flex-col gap-1 p-2">
        {navItems.map(({ page: itemPage, label, icon: Icon }) => (
          <button
            key={itemPage}
            onClick={() => onNavigate(itemPage)}
            title={label}
            className={cn(
              'flex items-center justify-center gap-3 rounded-md px-0 py-2 text-sm transition-colors md:justify-start md:px-3',
              page === itemPage
                ? 'bg-accent font-medium text-accent-foreground'
                : 'text-muted-foreground hover:bg-accent/50 hover:text-foreground',
            )}
          >
            <Icon className="size-4 shrink-0" />
            <span className="hidden md:inline">{label}</span>
          </button>
        ))}
      </nav>
      <div className="border-t p-2">
        <Button
          variant="ghost"
          className="w-full justify-center gap-3 px-0 text-muted-foreground md:justify-start md:px-3"
          disabled={!cfg || setConfig.isPending}
          onClick={() => cfg && setConfig.mutate({ ...cfg, mask_emails: !masked })}
          title="Mask email addresses across the app"
        >
          {masked ? <EyeOff className="size-4 shrink-0" /> : <Eye className="size-4 shrink-0" />}
          <span className="hidden md:inline">{masked ? 'Unmask emails' : 'Mask emails'}</span>
        </Button>
        <Button
          variant="ghost"
          className="w-full justify-center gap-3 px-0 text-muted-foreground md:justify-start md:px-3"
          onClick={() => setTheme(resolvedTheme === 'dark' ? 'light' : 'dark')}
          title="Toggle color theme"
        >
          {resolvedTheme === 'dark' ? (
            <Sun className="size-4 shrink-0" />
          ) : (
            <Moon className="size-4 shrink-0" />
          )}
          <span className="hidden md:inline">
            {resolvedTheme === 'dark' ? 'Light mode' : 'Dark mode'}
          </span>
        </Button>
      </div>
    </aside>
  )
}
