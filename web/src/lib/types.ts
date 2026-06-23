// Shared types mirroring the backend JSON shapes.

export type Role = "admin" | "user";
export type NetStatus = "pending" | "open" | "closed";

export interface User {
  id: string;
  callsign: string;
  firstName: string;
  lastName: string;
  email: string;
  role: Role;
  timezone: string;
  timeFormat: "24h" | "12h";
  createdAt: string;
  updatedAt: string;
}

export interface Net {
  id: string;
  name: string;
  netDate: string;
  ncsUserId: string;
  status: NetStatus;
  startAt: string | null;
  endAt: string | null;
  notes: string;
  createdAt: string;
  updatedAt: string;
  deletedAt?: string | null;
  ncsCallsign?: string;
}

export interface NetWithMeta extends Net {
  checkInCount: number;
}

export interface CheckIn {
  id: string;
  netId: string;
  callsign: string;
  nickname: string;
  hasTraffic: boolean;
  shortTime: boolean;
  notes: string;
  seq: number;
  checkedInAt: string;
  createdBy?: string | null;
  createdAt: string;
  updatedAt: string;
  deletedAt?: string | null;
}

export interface CallsignData {
  callsign: string;
  firstName: string;
  lastName: string;
  nickname: string;
  address1: string;
  address2: string;
  city: string;
  state: string;
  zip: string;
  country: string;
  dxcc: number | null;
  grid: string;
  latitude: number | null;
  longitude: number | null;
  cqZone: number | null;
  ituZone: number | null;
  iota: string;
  continent: string;
  email: string;
  website: string;
  qslManager: string;
  lotw: string;
  eqsl: string;
  flagIso2: string;
  source: string;
  lastLookupAt: string | null;
}

export interface Bootstrap {
  needsFirstAdmin: boolean;
  oidcEnabled: boolean;
  version: string;
  commit: string;
}

// Sync wire types.
export type SyncEntity = "net" | "checkin";
export type SyncOp = "put" | "delete";

export interface SyncChange {
  entity: SyncEntity;
  op: SyncOp;
  id: string;
  data?: unknown;
}

export interface SyncResult {
  id: string;
  entity: SyncEntity;
  status: "applied" | "conflict" | "error";
  message?: string;
  net?: Net;
  checkin?: CheckIn;
}

export interface PushResponse {
  serverTime: string;
  results: SyncResult[];
}

export interface PullResponse {
  serverTime: string;
  nets: Net[];
  checkins: CheckIn[];
}
