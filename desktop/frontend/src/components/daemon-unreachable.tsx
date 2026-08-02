import { PlugZap } from 'lucide-react'

import { Card, CardContent } from '@/components/ui/card'

/** Full-page state shown when the embedded monitoring runtime is not running. */
export function DaemonUnreachable() {
  return (
    <div className="flex h-full items-center justify-center p-6">
      <Card className="max-w-md">
        <CardContent className="flex flex-col items-center gap-3 p-8 text-center">
          <PlugZap className="size-8 text-muted-foreground" />
          <div className="font-medium">Monitor unavailable</div>
          <p className="text-sm text-muted-foreground">
            The built-in monitor runtime is not running, so no quota data is available.
          </p>
          <p className="text-xs text-muted-foreground">
            Restart Antigravity Token Monitor. The dashboard reconnects automatically when the
            runtime becomes available.
          </p>
        </CardContent>
      </Card>
    </div>
  )
}
