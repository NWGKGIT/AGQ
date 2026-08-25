import { ChevronRight } from "lucide-react";

import { QuotaBar } from "@/components/quota-bar";
import { Badge } from "@/components/ui/badge";
import { Card } from "@/components/ui/card";
import { ago, maskEmail, pct, until } from "@/lib/format";
import { deriveHealth, HEALTH_LABELS, type HealthStatus } from "@/lib/health";
import { groupByProvider, PROVIDER_COLORS } from "@/lib/providers";
import { cn } from "@/lib/utils";
import type { apiclient } from "../../../wailsjs/go/models";

const MODELS_SHOWN_PER_PROVIDER = 3;

const HEALTH_DOT: Record<HealthStatus, string> = {
  low: "bg-destructive",
  warning: "bg-warning",
  good: "bg-success",
  unknown: "bg-muted-foreground/40",
};

const HEALTH_BAR: Record<HealthStatus, string> = {
  low: "bg-destructive",
  warning: "bg-warning",
  good: "bg-success",
  unknown: "bg-muted-foreground/25",
};

function soonestReset(models: apiclient.ModelQuota[]): string | undefined {
  const times = models
    .map((m) => m.reset_time)
    .filter((t): t is string => !!t)
    .sort();
  return times[0];
}

/** Account summary with provider quotas and health state. */
export function AccountCard({
  account,
  live,
  masked,
  onSelect,
}: {
  account: apiclient.Account;
  live: boolean;
  masked: boolean;
  onSelect: (email: string) => void;
}) {
  const snapshot = account.latest_snapshot;
  const models = snapshot?.models ?? [];
  const health = deriveHealth(models);
  const groups = groupByProvider(models, (m) => m.label);
  const nextReset = soonestReset(models);

  return (
    <Card
      role="button"
      tabIndex={0}
      onClick={() => onSelect(account.email)}
      onKeyDown={(e) => {
        if (e.key === "Enter" || e.key === " ") {
          e.preventDefault();
          onSelect(account.email);
        }
      }}
      className={cn(
        "interactive-surface group flex cursor-pointer flex-col gap-3 overflow-hidden p-4 pb-0 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring",
        live && "border-sky-500/60 bg-sky-500/5 ring-1 ring-sky-500/20",
      )}
      aria-label={`${masked ? maskEmail(account.email) : account.email}, ${HEALTH_LABELS[health.status]} quota health, ${live ? "active account" : "idle"}`}
    >
      <div className="flex items-center justify-between gap-2">
        <span className="flex min-w-0 items-center gap-2">
          <span
            className={cn(
              "size-2 shrink-0 rounded-full",
              HEALTH_DOT[health.status],
            )}
            aria-hidden="true"
          />
          <span className="truncate font-mono text-sm">
            {masked ? maskEmail(account.email) : account.email}
          </span>
        </span>
        <div className="flex shrink-0 items-center gap-1.5">
          {live && (
            <Badge className="bg-sky-500 text-white hover:bg-sky-500">
              Current
            </Badge>
          )}
          <Badge
            variant={
              health.status === "low"
                ? "destructive"
                : health.status === "warning"
                  ? "warning"
                  : health.status === "good"
                    ? "success"
                    : "secondary"
            }
          >
            {HEALTH_LABELS[health.status]}
          </Badge>
          <ChevronRight
            className="size-3.5 text-muted-foreground/0 transition-colors duration-150 group-hover:text-muted-foreground"
            aria-hidden="true"
          />
        </div>
      </div>

      <div className="flex justify-between text-[11px] uppercase tracking-wide text-muted-foreground">
        <span>
          {snapshot
            ? `synced ${ago(snapshot.staleness_seconds)} ago`
            : "no snapshot"}
        </span>
        <span>
          {nextReset ? `resets ${until(nextReset)}` : account.plan_name || ""}
        </span>
      </div>

      <div className="h-px bg-border" />

      {groups.length === 0 ? (
        <p className="py-2 text-xs text-muted-foreground">
          No model quota data yet.
        </p>
      ) : (
        <div className="flex flex-col gap-3">
          {groups.map(({ provider, items }) => {
            const shown = items.slice(0, MODELS_SHOWN_PER_PROVIDER);
            const hidden = items.length - shown.length;
            return (
              <div key={provider}>
                <span className="mb-1.5 flex items-center gap-1.5 text-[10px] font-semibold uppercase tracking-widest text-muted-foreground/70">
                  <span
                    className="size-1.5 rounded-full"
                    style={{ backgroundColor: PROVIDER_COLORS[provider] }}
                    aria-hidden="true"
                  />
                  {provider}
                </span>
                <div className="flex flex-col gap-1.5">
                  {shown.map((m) => (
                    <div
                      key={m.model_id + m.label}
                      className="flex items-center gap-2 text-xs"
                    >
                      <span
                        className="w-24 truncate text-muted-foreground"
                        title={m.label}
                      >
                        {m.label}
                      </span>
                      <QuotaBar
                        fraction={m.remaining_fraction}
                        className="h-[2px] flex-1"
                      />
                      <span className="tnum w-9 text-right font-mono">
                        {pct(m.remaining_fraction)}
                      </span>
                      <span className="tnum w-12 text-right font-mono text-muted-foreground">
                        {m.reset_time ? until(m.reset_time) : "–"}
                      </span>
                    </div>
                  ))}
                  {hidden > 0 && (
                    <span className="text-[10px] text-muted-foreground">
                      +{hidden} more
                    </span>
                  )}
                </div>
              </div>
            );
          })}
        </div>
      )}

      <div className="-mx-4 mt-auto h-[3px] bg-muted" aria-hidden="true">
        <div
          className={cn(
            "h-full transition-all duration-300",
            HEALTH_BAR[health.status],
          )}
          style={{
            width: `${Math.round((health.lowestRemainingFraction ?? 0) * 100)}%`,
          }}
        />
      </div>
    </Card>
  );
}
