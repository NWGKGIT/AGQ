import { useEffect, useState } from 'react'
import { Loader2 } from 'lucide-react'

import { useTheme, type Theme } from '@/components/theme-provider'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Input } from '@/components/ui/input'
import { Switch } from '@/components/ui/switch'
import { useAppConfig, useDaemonStatus, useSetAppConfig } from '@/lib/api'
import { ago, shortDateTime, until } from '@/lib/format'
import { cn } from '@/lib/utils'

function Row({
  label,
  description,
  children,
}: {
  label: string
  description?: string
  children: React.ReactNode
}) {
  return (
    <div className="flex items-center justify-between gap-6 py-3">
      <div>
        <div className="text-sm font-medium">{label}</div>
        {description && <div className="mt-0.5 text-xs text-muted-foreground">{description}</div>}
      </div>
      {children}
    </div>
  )
}

function LocalApiSection() {
  const { data: cfg } = useAppConfig()
  const setConfig = useSetAppConfig()
  const [port, setPort] = useState('')

  useEffect(() => {
    if (cfg) setPort(String(cfg.port))
  }, [cfg])

  const exposed = cfg?.expose_api ?? false
  const parsed = Number.parseInt(port, 10)
  const valid = Number.isInteger(parsed) && parsed > 0 && parsed <= 65535
  const dirty = cfg != null && valid && parsed !== cfg.port

  const save = () => {
    if (!cfg || !valid) return
    setConfig.mutate({ ...cfg, port: parsed })
  }

  return (
    <Card>
      <CardHeader>
        <CardTitle className="text-base">Local API</CardTitle>
        <CardDescription>
          The app talks to its built-in monitor directly. Optionally expose the monitor's JSON API
          on localhost for scripts and other tools.
        </CardDescription>
      </CardHeader>
      <CardContent className="divide-y">
        <Row
          label="Expose local API"
          description="Serves the read-only API on 127.0.0.1 for external tools (curl, scripts)."
        >
          <Switch
            checked={exposed}
            disabled={!cfg || setConfig.isPending}
            onCheckedChange={(checked) => cfg && setConfig.mutate({ ...cfg, expose_api: checked })}
          />
        </Row>
        {exposed && (
          <Row label="API port" description="Localhost only; the monitor restarts after saving.">
            <div className="flex items-center gap-2">
              <Input
                value={port}
                onChange={(e) => setPort(e.target.value.replace(/\D/g, ''))}
                inputMode="numeric"
                className={cn('w-24 font-mono', !valid && port !== '' && 'border-destructive')}
              />
              <Button size="sm" onClick={save} disabled={!dirty || setConfig.isPending}>
                {setConfig.isPending ? <Loader2 className="animate-spin" /> : 'Save'}
              </Button>
            </div>
          </Row>
        )}
      </CardContent>
    </Card>
  )
}

function AppearanceSection() {
  const { theme, setTheme } = useTheme()
  const options: Theme[] = ['light', 'dark', 'system']

  return (
    <Card>
      <CardHeader>
        <CardTitle className="text-base">Appearance</CardTitle>
      </CardHeader>
      <CardContent>
        <Row label="Theme">
          <div className="flex overflow-hidden rounded-md border text-xs">
            {options.map((option) => (
              <button
                key={option}
                onClick={() => setTheme(option)}
                className={cn(
                  'px-3 py-1.5 capitalize transition-colors',
                  option === theme
                    ? 'bg-primary text-primary-foreground'
                    : 'text-muted-foreground hover:text-foreground',
                )}
              >
                {option}
              </button>
            ))}
          </div>
        </Row>
      </CardContent>
    </Card>
  )
}

function PrivacySection() {
  const { data: cfg } = useAppConfig()
  const setConfig = useSetAppConfig()

  return (
    <Card>
      <CardHeader>
        <CardTitle className="text-base">Privacy</CardTitle>
      </CardHeader>
      <CardContent>
        <Row
          label="Mask emails"
          description="Shows j***@gmail.com everywhere — useful for screenshots and screen shares."
        >
          <Switch
            checked={cfg?.mask_emails ?? false}
            disabled={!cfg || setConfig.isPending}
            onCheckedChange={(checked) => cfg && setConfig.mutate({ ...cfg, mask_emails: checked })}
          />
        </Row>
      </CardContent>
    </Card>
  )
}

function MonitorSection() {
  const { data: status, error } = useDaemonStatus()

  return (
    <Card>
      <CardHeader>
        <CardTitle className="text-base">Monitor runtime</CardTitle>
        <CardDescription>Local quota collection and account detection status.</CardDescription>
      </CardHeader>
      <CardContent>
        {error ? (
          <div className="space-y-2 py-2 text-sm">
            <p className="text-destructive">The monitor runtime is not reachable.</p>
            <p className="text-xs text-muted-foreground">
              Restart Antigravity Token Monitor. It will reconnect automatically when the local
              runtime becomes available.
            </p>
          </div>
        ) : status ? (
          <dl className="grid grid-cols-2 gap-x-6 gap-y-2 py-2 text-sm">
            <dt className="text-muted-foreground">State</dt>
            <dd className="font-mono">{status.state}</dd>
            <dt className="text-muted-foreground">Uptime</dt>
            <dd className="font-mono">{status.uptime}</dd>
            <dt className="text-muted-foreground">Started</dt>
            <dd className="font-mono">{shortDateTime(status.started_at)}</dd>
            {status.last_poll_at && (
              <>
                <dt className="text-muted-foreground">Last poll</dt>
                <dd className="font-mono">
                  {ago((Date.now() - new Date(status.last_poll_at).getTime()) / 1000)} ago
                </dd>
              </>
            )}
            {status.next_poll_at && (
              <>
                <dt className="text-muted-foreground">Next poll</dt>
                <dd className="font-mono">in {until(status.next_poll_at)}</dd>
              </>
            )}
          </dl>
        ) : (
          <p className="py-2 text-sm text-muted-foreground">Loading…</p>
        )}
      </CardContent>
    </Card>
  )
}

export function SettingsPage() {
  return (
    <div className="mx-auto flex max-w-2xl flex-col gap-5 p-6">
      <header className="border-b pb-4">
        <h1 className="text-lg font-semibold">Settings</h1>
        <p className="mt-0.5 text-sm text-muted-foreground">
          Local API, appearance, and privacy.
        </p>
      </header>
      <LocalApiSection />
      <AppearanceSection />
      <PrivacySection />
      <MonitorSection />
    </div>
  )
}
