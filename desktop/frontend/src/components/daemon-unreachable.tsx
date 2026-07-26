import { PlugZap } from 'lucide-react'

import { Card, CardContent } from '@/components/ui/card'

/** Full-page state shown when the local monitoring runtime cannot be reached. */
export function DaemonUnreachable({ port }: { port?: number }) {
  return (
    <div className="flex h-full items-center justify-center p-6">
      <Card className="max-w-md">
        <CardContent className="flex flex-col items-center gap-3 p-8 text-center">
          <PlugZap className="size-8 text-muted-foreground" />
          <div className="font-medium">Monitor unavailable</div>
          <p className="text-sm text-muted-foreground">
            Antigravity Token Monitor cannot reach its local data source on{' '}
            <span className="font-mono">localhost:{port ?? 7432}</span>.
          </p>
          <p className="text-xs text-muted-foreground">
            Check the monitor runtime in Settings. The dashboard reconnects automatically when
            the local data source is available.
          </p>
        </CardContent>
      </Card>
    </div>
  )
}
