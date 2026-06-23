// Calendar-date + duration helpers. Timestamp display (UTC + local, 12/24h)
// lives in datetime.ts; this file is only the date-only and elapsed helpers.

export function formatDate(iso: string | null | undefined): string {
  if (!iso) return "—";
  const d = new Date(iso + "T00:00:00");
  if (isNaN(d.getTime())) return iso ?? "—";
  return d.toLocaleDateString(undefined, {
    year: "numeric",
    month: "short",
    day: "2-digit",
  });
}

// elapsed renders a compact running duration between two instants.
export function elapsed(
  startIso: string | null,
  endIso: string | null,
): string {
  if (!startIso) return "—";
  const start = new Date(startIso).getTime();
  const end = endIso ? new Date(endIso).getTime() : Date.now();
  let secs = Math.max(0, Math.floor((end - start) / 1000));
  const h = Math.floor(secs / 3600);
  secs -= h * 3600;
  const m = Math.floor(secs / 60);
  if (h > 0) return `${h}h ${m}m`;
  return `${m}m`;
}

export function todayDate(): string {
  return new Date().toISOString().slice(0, 10);
}
