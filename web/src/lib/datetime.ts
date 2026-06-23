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
    base.day = "2-digit";
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

// dualTime is the time-only variant (e.g. per check-in rows).
export function dualTime(iso: string | null | undefined): string {
  return dual(iso, false, false);
}
