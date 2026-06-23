// Local-first storage. The app writes here first; a background flusher syncs the
// outbox to the backend. Reads in the UI come from Dexie via liveQuery so the
// interface works the same online or offline.
import Dexie, { type EntityTable } from "dexie";
import type { Net, CheckIn, CallsignData } from "./types";

// OutboxEntry is a pending local change awaiting sync to the backend.
export interface OutboxEntry {
  localId?: number;
  entity: "net" | "checkin";
  op: "put" | "delete";
  entityId: string;
  // payload is the field set to send (server fills authoritative fields).
  payload: Record<string, unknown>;
  createdAt: string;
  attempts: number;
}

export interface MetaEntry {
  key: string;
  value: string;
}

export const db = new Dexie("netlog") as Dexie & {
  nets: EntityTable<Net, "id">;
  checkins: EntityTable<CheckIn, "id">;
  callsigns: EntityTable<CallsignData, "callsign">;
  outbox: EntityTable<OutboxEntry, "localId">;
  meta: EntityTable<MetaEntry, "key">;
};

db.version(1).stores({
  nets: "&id, status, updatedAt, endAt",
  checkins: "&id, netId, callsign, updatedAt",
  callsigns: "&callsign",
  outbox: "++localId, entityId, createdAt",
  meta: "&key",
});

const LAST_SYNC_KEY = "lastSyncTime";

export async function getLastSync(): Promise<string> {
  const row = await db.meta.get(LAST_SYNC_KEY);
  return row?.value ?? "";
}

export async function setLastSync(value: string): Promise<void> {
  await db.meta.put({ key: LAST_SYNC_KEY, value });
}

// resetLocal clears all local data (used on logout to avoid leaking another
// operator's logs when a different account signs in on the same browser).
export async function resetLocal(): Promise<void> {
  await Promise.all([
    db.nets.clear(),
    db.checkins.clear(),
    db.callsigns.clear(),
    db.outbox.clear(),
    db.meta.clear(),
  ]);
}
