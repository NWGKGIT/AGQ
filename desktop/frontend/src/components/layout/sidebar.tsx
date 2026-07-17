import { BarChart3, Eye, EyeOff, LayoutDashboard, Moon, Settings, Sun } from 'lucide-react'

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
    <aside className="flex h-full w-52 shrink-0 flex-col border-r bg-card">
      <div className="flex h-14 items-center gap-2 border-b px-4">
        <span className="font-mono text-lg font-bold tracking-tighter">AGQ</span>
        <span className="text-xs text-muted-foreground">quota monitor</span>
      </div>
      <nav className="flex flex-1 flex-col gap-1 p-2">
        {navItems.map(({ page: itemPage, label, icon: Icon }) => (
          <button
            key={itemPage}
            onClick={() => onNavigate(itemPage)}
            className={cn(
              'flex items-center gap-3 rounded-md px-3 py-2 text-sm transition-colors',
              page === itemPage
                ? 'bg-accent font-medium text-accent-foreground'
                : 'text-muted-foreground hover:bg-accent/50 hover:text-foreground',
            )}
          >
            <Icon className="size-4" />
            {label}
          </button>
        ))}
      </nav>
      <div className="border-t p-2">
        <Button
          variant="ghost"
          className="w-full justify-start gap-3 px-3 text-muted-foreground"
          disabled={!cfg || setConfig.isPending}
          onClick={() => cfg && setConfig.mutate({ ...cfg, mask_emails: !masked })}
          title="Mask email addresses across the app"
        >
          {masked ? <EyeOff className="size-4" /> : <Eye className="size-4" />}
          {masked ? 'Unmask emails' : 'Mask emails'}
        </Button>
        <Button
          variant="ghost"
          className="w-full justify-start gap-3 px-3 text-muted-foreground"
          onClick={() => setTheme(resolvedTheme === 'dark' ? 'light' : 'dark')}
        >
          {resolvedTheme === 'dark' ? <Sun className="size-4" /> : <Moon className="size-4" />}
          {resolvedTheme === 'dark' ? 'Light mode' : 'Dark mode'}
        </Button>
      </div>
    </aside>
  )
}
