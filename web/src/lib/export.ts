// Net export. CSV is generated entirely client-side (works offline). "PDF" opens
// a clean, light-themed printable document and invokes the browser's print
// dialog, where the operator can Save as PDF — no server round-trip or PDF
// library needed.
import type { CallsignData, CheckIn, Net, User } from "./types";

function utc(iso: string | null | undefined): string {
  if (!iso) return "";
  const d = new Date(iso);
  if (isNaN(d.getTime())) return "";
  const p = (n: number) => String(n).padStart(2, "0");
  return (
    `${d.getUTCFullYear()}-${p(d.getUTCMonth() + 1)}-${p(d.getUTCDate())} ` +
    `${p(d.getUTCHours())}:${p(d.getUTCMinutes())}:${p(d.getUTCSeconds())}Z`
  );
}

function fullName(c?: CallsignData): string {
  return c ? [c.firstName, c.lastName].filter(Boolean).join(" ") : "";
}

function safeFile(name: string): string {
  return name.replace(/[^a-z0-9]+/gi, "_").replace(/^_+|_+$/g, "") || "net";
}

function escapeCSV(v: string): string {
  // Neutralize spreadsheet formula injection: a cell beginning with = + - @ (or
  // a leading control char) is treated as a formula by Excel/Sheets/LibreOffice.
  // Prefixing with an apostrophe forces it to be read as text. Callbook fields
  // (untrusted) and user input both flow through here.
  if (/^[=+\-@\t\r]/.test(v)) v = "'" + v;
  if (/[",\n]/.test(v)) return `"${v.replace(/"/g, '""')}"`;
  return v;
}

// exportNetCSV builds and downloads a CSV of the net's check-ins.
export function exportNetCSV(
  net: Net,
  checkins: CheckIn[],
  calls: Map<string, CallsignData>,
): void {
  const header = [
    "Seq",
    "Callsign",
    "Name",
    "Nickname",
    "Country",
    "Has Traffic",
    "Short Time",
    "Checked In (UTC)",
  ];
  const rows = checkins.map((c) => {
    const cb = calls.get(c.callsign);
    return [
      String(c.seq),
      c.callsign,
      fullName(cb),
      c.nickname,
      cb?.country ?? "",
      c.hasTraffic ? "Yes" : "No",
      c.shortTime ? "Yes" : "No",
      utc(c.checkedInAt),
    ];
  });

  const meta = [
    ["Net", net.name],
    ["Date", net.netDate],
    ["NCS", net.ncsCallsign ?? ""],
    ["Started (UTC)", utc(net.startAt)],
    ["Ended (UTC)", utc(net.endAt)],
    ["Check-ins", String(checkins.length)],
  ];

  const lines: string[] = [];
  for (const [k, v] of meta) lines.push(`${escapeCSV(k)},${escapeCSV(v)}`);
  lines.push("");
  lines.push(header.map(escapeCSV).join(","));
  for (const r of rows) lines.push(r.map(escapeCSV).join(","));
  if (net.notes) {
    lines.push("");
    lines.push(`${escapeCSV("Notes")},${escapeCSV(net.notes)}`);
  }

  const blob = new Blob([lines.join("\n")], { type: "text/csv;charset=utf-8" });
  const url = URL.createObjectURL(blob);
  const a = document.createElement("a");
  a.href = url;
  a.download = `${safeFile(net.name)}_${net.netDate}.csv`;
  document.body.appendChild(a);
  a.click();
  a.remove();
  URL.revokeObjectURL(url);
}

function esc(s: string): string {
  return s.replace(/[&<>"]/g, (ch) => {
    switch (ch) {
      case "&":
        return "&amp;";
      case "<":
        return "&lt;";
      case ">":
        return "&gt;";
      default:
        return "&quot;";
    }
  });
}

// exportNetPDF opens a print-ready, light-themed document and triggers print.
export function exportNetPDF(
  net: Net,
  checkins: CheckIn[],
  calls: Map<string, CallsignData>,
  user: User | null,
): void {
  const rows = checkins
    .map((c) => {
      const cb = calls.get(c.callsign);
      const tags = [c.hasTraffic ? "Traffic" : "", c.shortTime ? "Short" : ""]
        .filter(Boolean)
        .join(", ");
      return `<tr>
        <td class="num">${c.seq}</td>
        <td class="call">${esc(c.callsign)}</td>
        <td>${esc(fullName(cb))}</td>
        <td>${esc(c.nickname)}</td>
        <td>${esc(cb?.country ?? "")}</td>
        <td>${esc(tags)}</td>
        <td class="num">${esc(utc(c.checkedInAt))}</td>
      </tr>`;
    })
    .join("");

  const notesBlock = net.notes
    ? `<h2>Notes</h2><p class="notes">${esc(net.notes)}</p>`
    : "";

  const html = `<!doctype html>
<html lang="en"><head><meta charset="utf-8" />
<title>${esc(net.name)} — ${esc(net.netDate)}</title>
<style>
  :root { color-scheme: light; }
  * { box-sizing: border-box; }
  body { font-family: ui-sans-serif, system-ui, -apple-system, Segoe UI, Roboto, sans-serif;
         color: #18181b; background: #fff; margin: 32px; }
  h1 { font-size: 20px; margin: 0 0 2px; }
  .sub { color: #71717a; font-size: 12px; margin: 0 0 16px; }
  .meta { display: grid; grid-template-columns: repeat(4, 1fr); gap: 8px 16px; margin-bottom: 20px;
          font-size: 12px; }
  .meta dt { color: #71717a; text-transform: uppercase; letter-spacing: .04em; font-size: 10px; }
  .meta dd { margin: 0; font-variant-numeric: tabular-nums; }
  table { width: 100%; border-collapse: collapse; font-size: 12px; }
  th, td { text-align: left; padding: 6px 8px; border-bottom: 1px solid #e4e4e7; }
  th { font-size: 10px; text-transform: uppercase; letter-spacing: .04em; color: #71717a; }
  .num { text-align: right; font-variant-numeric: tabular-nums; }
  .call { font-family: ui-monospace, SFMono-Regular, Menlo, monospace; font-weight: 600; }
  h2 { font-size: 13px; text-transform: uppercase; letter-spacing: .04em; color: #71717a;
       margin: 24px 0 6px; }
  .notes { white-space: pre-wrap; font-size: 12px; }
  .footer { margin-top: 24px; color: #a1a1aa; font-size: 10px; }
  @media print { body { margin: 16px; } }
</style></head>
<body>
  <h1>${esc(net.name)}</h1>
  <p class="sub">Directed net log${user ? ` · exported by ${esc(user.callsign)}` : ""}</p>
  <dl class="meta">
    <div><dt>Date</dt><dd>${esc(net.netDate)}</dd></div>
    <div><dt>NCS</dt><dd>${esc(net.ncsCallsign ?? "")}</dd></div>
    <div><dt>Started (UTC)</dt><dd>${esc(utc(net.startAt))}</dd></div>
    <div><dt>Ended (UTC)</dt><dd>${esc(utc(net.endAt))}</dd></div>
  </dl>
  <table>
    <thead><tr>
      <th class="num">#</th><th>Callsign</th><th>Name</th><th>Nickname</th>
      <th>Country</th><th>Flags</th><th class="num">Checked In (UTC)</th>
    </tr></thead>
    <tbody>${rows || `<tr><td colspan="7">No check-ins.</td></tr>`}</tbody>
  </table>
  ${notesBlock}
  <p class="footer">${checkins.length} check-in(s) · Generated by NetLog</p>
  <script>window.onload = function(){ window.print(); };<\/script>
</body></html>`;

  // Open via a Blob URL (no document.write). All interpolated values are escaped.
  const blob = new Blob([html], { type: "text/html" });
  const url = URL.createObjectURL(blob);
  const win = window.open(url, "_blank");
  // Revoke after the new document has had time to load and print.
  setTimeout(() => URL.revokeObjectURL(url), 60_000);
  if (!win) URL.revokeObjectURL(url);
}
