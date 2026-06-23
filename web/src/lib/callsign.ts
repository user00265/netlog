// Single source of truth for client-side callsign normalization, mirroring the
// server's validate.NormalizeCallsign (uppercase + trim). The backend
// re-normalizes regardless; this keeps optimistic local state consistent.
export function normalizeCallsign(s: string): string {
  return s.toUpperCase().trim();
}
