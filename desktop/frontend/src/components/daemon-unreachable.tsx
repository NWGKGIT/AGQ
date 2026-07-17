import { PlugZap } from 'lucide-react'

import { Card, CardContent } from '@/components/ui/card'

/** Full-page state shown when the local daemon cannot be reached. */
export function DaemonUnreachable({ port }: { port?: number }) {
  return (
    <div className="flex h-full items-center justify-center p-6">
      <Card className="max-w-md">
        <CardContent className="flex flex-col items-center gap-3 p-8 text-center">
          <PlugZap className="size-8 text-muted-foreground" />
          <div className="font-medium">Daemon unreachable</div>
          <p className="text-sm text-muted-foreground">
            AGQ Daemon is not responding on{' '}
            <span className="font-mono">localhost:{port ?? 7432}</span>. Start it with:
          </p>
          <code className="select-text rounded-md bg-muted px-3 py-1.5 font-mono text-xs">
            systemctl --user start agq
          </code>
          <p className="text-xs text-muted-foreground">
            or run <span className="font-mono">make run</span> from the repository. The dashboard
            reconnects automatically.
          </p>
        </CardContent>
      </Card>
    </div>
  )
}
