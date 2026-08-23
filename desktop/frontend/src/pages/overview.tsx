import { useMemo, useState } from "react";
import { Inbox } from "lucide-react";

import { DaemonUnreachable } from "@/components/daemon-unreachable";
import { AccountCard } from "@/components/overview/account-card";
import { AccountSheet } from "@/components/overview/account-sheet";
import { HealthSummary } from "@/components/overview/health-summary";
import { ProviderStrip } from "@/components/overview/provider-strip";
import { StatusHeader } from "@/components/overview/status-header";
import { Skeleton } from "@/components/ui/skeleton";
import {
  isDaemonUnreachable,
  useAccounts,
  useAppConfig,
  useCurrentAccount,
} from "@/lib/api";
import { deriveHealth, HEALTH_SORT_ORDER } from "@/lib/health";

export function OverviewPage() {
  const { data: cfg } = useAppConfig();
  const { data: accountsData, isPending, error } = useAccounts();
  const { data: current } = useCurrentAccount();
  const [selectedEmail, setSelectedEmail] = useState<string | null>(null);

  const masked = cfg?.mask_emails ?? false;
  const liveEmails = useMemo(
    () => new Set(current?.is_live ? current.accounts.map((a) => a.email) : []),
    [current],
  );
  const accounts = useMemo(
    () =>
      [...(accountsData?.accounts ?? [])].sort((a, b) => {
        const activeDifference =
          Number(liveEmails.has(a.email)) - Number(liveEmails.has(b.email));
        if (activeDifference !== 0) return -activeDifference;

        const healthDifference =
          HEALTH_SORT_ORDER[
            deriveHealth(a.latest_snapshot?.models ?? []).status
          ] -
          HEALTH_SORT_ORDER[
            deriveHealth(b.latest_snapshot?.models ?? []).status
          ];
        if (healthDifference !== 0) return healthDifference;
        return b.last_seen.localeCompare(a.last_seen);
      }),
    [accountsData, liveEmails],
  );
  const selected = accounts.find((a) => a.email === selectedEmail) ?? null;

  if (error && isDaemonUnreachable(error)) {
    return <DaemonUnreachable />;
  }

  return (
    <div className="mx-auto flex max-w-6xl flex-col gap-5 p-6">
      <header className="flex items-end justify-between border-b pb-4">
        <div>
          <h1 className="text-lg font-semibold">Overview</h1>
          <p className="mt-0.5 text-sm text-muted-foreground">
            Quota usage across every observed account.
          </p>
        </div>
      </header>

      <StatusHeader masked={masked} />
      <HealthSummary accounts={accounts} />
      <ProviderStrip />

      {isPending ? (
        <div className="grid grid-cols-1 gap-3 md:grid-cols-2 xl:grid-cols-3">
          {Array.from({ length: 6 }, (_, i) => (
            <Skeleton key={i} className="h-48" />
          ))}
        </div>
      ) : error ? (
        <p className="py-8 text-center text-sm text-destructive">
          {String(error)}
        </p>
      ) : accounts.length === 0 ? (
        <div className="flex flex-col items-center gap-2 py-16 text-muted-foreground">
          <Inbox className="size-8" />
          <p className="text-sm">No accounts observed yet.</p>
          <p className="max-w-sm text-center text-xs">
            Log in to Antigravity and the monitor will record quota snapshots
            within a minute.
          </p>
        </div>
      ) : (
        <div className="grid grid-cols-1 gap-3 md:grid-cols-2 xl:grid-cols-3">
          {accounts.map((account) => (
            <AccountCard
              key={account.id}
              account={account}
              live={liveEmails.has(account.email)}
              masked={masked}
              onSelect={setSelectedEmail}
            />
          ))}
        </div>
      )}

      <AccountSheet
        account={selected}
        masked={masked}
        onClose={() => setSelectedEmail(null)}
      />
    </div>
  );
}
