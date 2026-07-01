// Time display. The backend stores UTC; the operator sees UTC first, then their
// own local time in parentheses, in 24h (default) or 12h format per their
// preference. The NCS timezone comes from their profile, falling back to the
// browser's zone.
import { auth } from "./stores/auth.svelte";

// browserTimezone returns the runtime's IANA zone, falling back to UTC.
export function browserTimezone(): string {
  try {
    return Intl.DateTimeFormat().resolvedOptions().timeZone || "UTC";
  } catch {
    return "UTC";
  }
}

export function userTimezone(): string {
  return auth.user?.timezone || browserTimezone();
}

function hour12(): boolean {
  return (auth.user?.timeFormat ?? "24h") === "12h";
}

function parts(withDate: boolean): Intl.DateTimeFormatOptions {
  const base: Intl.DateTimeFormatOptions = {
    hour: "2-digit",
    minute: "2-digit",
    hour12: hour12(),
  };
  if (withDate) {
    base.year = "numeric";
    base.month = "short";
    base.day = "numeric";
  }
  return base;
}

function fmt(d: Date, tz: string, withDate: boolean): string {
  return new Intl.DateTimeFormat(undefined, {
    ...parts(withDate),
    timeZone: tz,
  }).format(d);
}

// dual renders "UTC (local)". The UTC side carries the date; the local side is
// time-only to stay compact (set localDate to include the local calendar date).
export function dual(
  iso: string | null | undefined,
  withDate = true,
  localDate = false,
): string {
  if (!iso) return "—";
  const d = new Date(iso);
  if (isNaN(d.getTime())) return "—";
  // "Z" is the idiomatic, compact UTC marker for operators and keeps columns tight.
  const utc = `${fmt(d, "UTC", withDate)}Z`;
  const local = fmt(d, userTimezone(), localDate);
  return `${utc} (${local})`;
}

// zuluWithDate renders "Jun 28, 2026 0148Z" — UTC date + compact HHMM time,
// no local side, no colon in the time. Used in list views where the local
// timezone adds noise and the compact zulu time is familiar to operators.
export function zuluWithDate(iso: string | null | undefined): string {
  if (!iso) return "—";
  const d = new Date(iso);
  if (isNaN(d.getTime())) return "—";
  const date = new Intl.DateTimeFormat("en-US", {
    month: "short",
    day: "numeric",
    year: "numeric",
    timeZone: "UTC",
  }).format(d);
  const time = new Intl.DateTimeFormat("en-GB", {
    hour: "2-digit",
    minute: "2-digit",
    hour12: false,
    timeZone: "UTC",
  }).format(d);
  return `${date} ${time.replace(":", "")}Z`;
}

// zulu renders a compact UTC time stamp, e.g. "0130Z". Amateur-radio timestamps
// are conventionally 24h Zulu, so this ignores the 12h preference and shows no
// date or local time — used per check-in row.
export function zulu(iso: string | null | undefined): string {
  if (!iso) return "—";
  const d = new Date(iso);
  if (isNaN(d.getTime())) return "—";
  const time = new Intl.DateTimeFormat("en-GB", {
    hour: "2-digit",
    minute: "2-digit",
    hour12: false,
    timeZone: "UTC",
  }).format(d); // "01:30"
  return `${time.replace(":", "")}Z`;
}
