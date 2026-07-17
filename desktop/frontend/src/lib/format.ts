/** Formatting helpers shared across pages. */

/** Formats a fraction (0..1) as a whole percent, or a placeholder when null. */
export function pct(fraction: number | null | undefined): string {
  if (fraction == null) return '–'
  return `${Math.round(fraction * 100)}%`
}

/** Formats seconds-ago as a compact relative time: 12s, 4m, 2h, 3d. */
export function ago(seconds: number): string {
  if (seconds < 0) seconds = 0
  if (seconds < 60) return `${Math.floor(seconds)}s`
  if (seconds < 3600) return `${Math.floor(seconds / 60)}m`
  if (seconds < 86400) return `${Math.floor(seconds / 3600)}h`
  return `${Math.floor(seconds / 86400)}d`
}

/** Formats a future instant as a compact countdown: 12m, 4h 22m, 3d 4h. */
export function until(iso: string | null | undefined, now = new Date()): string {
  if (!iso) return '–'
  const ms = new Date(iso).getTime() - now.getTime()
  if (Number.isNaN(ms)) return '–'
  if (ms <= 0) return 'now'
  const minutes = Math.floor(ms / 60000)
  if (minutes < 60) return `${minutes}m`
  const hours = Math.floor(minutes / 60)
  if (hours < 24) return `${hours}h ${minutes % 60}m`
  return `${Math.floor(hours / 24)}d ${hours % 24}h`
}

/** Formats an ISO timestamp as a short local date-time, e.g. "Jul 1, 14:22". */
export function shortDateTime(iso: string): string {
  const d = new Date(iso)
  if (Number.isNaN(d.getTime())) return iso
  return d.toLocaleString(undefined, {
    month: 'short',
    day: 'numeric',
    hour: '2-digit',
    minute: '2-digit',
  })
}

/** Formats an ISO timestamp as local time only, e.g. "14:22:01". */
export function timeOnly(iso: string): string {
  const d = new Date(iso)
  if (Number.isNaN(d.getTime())) return iso
  return d.toLocaleTimeString(undefined, {
    hour: '2-digit',
    minute: '2-digit',
    second: '2-digit',
  })
}

/** Masks an email's local part: "j.doe@gmail.com" -> "j***@gmail.com". */
export function maskEmail(email: string): string {
  const at = email.indexOf('@')
  if (at <= 0) return email
  return `${email[0]}***${email.slice(at)}`
}
