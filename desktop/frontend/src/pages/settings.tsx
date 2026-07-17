import { useEffect, useState } from 'react'
import { CheckCircle2, Loader2, XCircle } from 'lucide-react'

import { useTheme, type Theme } from '@/components/theme-provider'
import { Button } from '@/components/ui/button'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Input } from '@/components/ui/input'
import { Switch } from '@/components/ui/switch'
import { useAppConfig, useDaemonStatus, useSetAppConfig } from '@/lib/api'
import { ago, shortDateTime, until } from '@/lib/format'
import { cn } from '@/lib/utils'
import { GetHealth } from '../../wailsjs/go/main/App'

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

function ConnectionSection() {
  const { data: cfg } = useAppConfig()
  const setConfig = useSetAppConfig()
  const [port, setPort] = useState('')
  const [test, setTest] = useState<'idle' | 'testing' | 'ok' | 'failed'>('idle')

  useEffect(() => {
    if (cfg) setPort(String(cfg.port))
  }, [cfg])

  const parsed = Number.parseInt(port, 10)
  const valid = Number.isInteger(parsed) && parsed > 0 && parsed <= 65535
  const dirty = cfg != null && valid && parsed !== cfg.port

  const testConnection = async () => {
    setTest('testing')
    try {
      // Tests the *saved* port; save first when the field was changed.
      await GetHealth()
      setTest('ok')
    } catch {
      setTest('failed')
    }
  }

  const save = () => {
    if (!cfg || !valid) return
    setTest('idle')
    setConfig.mutate({ ...cfg, port: parsed })
  }

  return (
    <Card>
      <CardHeader>
        <CardTitle className="text-base">Daemon connection</CardTitle>
        <CardDescription>
          Where the AGQ daemon serves its local API. Matches the daemon's{' '}
          <span className="font-mono text-xs">AGQ_PORT</span> (default 7432).
        </CardDescription>
      </CardHeader>
      <CardContent className="divide-y">
        <Row label="API port" description="localhost only; the app reconnects after saving.">
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
        <Row label="Connection test" description="Calls /api/health on the saved port.">
          <div className="flex items-center gap-2">
            {test === 'ok' && (
              <span className="flex items-center gap-1 text-xs text-success">
                <CheckCircle2 className="size-3.5" /> reachable
              </span>
            )}
            {test === 'failed' && (
              <span className="flex items-center gap-1 text-xs text-destructive">
                <XCircle className="size-3.5" /> unreachable
              </span>
            )}
            <Button size="sm" variant="outline" onClick={testConnection} disabled={test === 'testing'}>
              {test === 'testing' ? <Loader2 className="animate-spin" /> : 'Test'}
            </Button>
          </div>
        </Row>
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

function DaemonSection() {
  const { data: status, error } = useDaemonStatus()

  return (
    <Card>
      <CardHeader>
        <CardTitle className="text-base">Daemon</CardTitle>
        <CardDescription>Runs as a systemd user service, independent of this app.</CardDescription>
      </CardHeader>
      <CardContent>
        {error ? (
          <div className="space-y-2 py-2 text-sm">
            <p className="text-destructive">Not reachable.</p>
            <code className="block w-fit select-text rounded-md bg-muted px-3 py-1.5 font-mono text-xs">
              systemctl --user start agq
            </code>
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
          Connection, appearance, and privacy.
        </p>
      </header>
      <ConnectionSection />
      <AppearanceSection />
      <PrivacySection />
      <DaemonSection />
    </div>
  )
}
