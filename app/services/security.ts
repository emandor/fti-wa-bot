import { Database } from "bun:sqlite";
import { rateLimit } from "elysia-rate-limit";

const db = new Database("logs.db");

db.query(
  `
  CREATE TABLE IF NOT EXISTS unauthorized_logs (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    ip TEXT,
    reason TEXT,
    headers TEXT,
    timestamp DATETIME DEFAULT CURRENT_TIMESTAMP
  )
`,
).run();

export function logUnauthorizedAttempt({
  ip,
  reason,
  headers,
}: {
  ip?: string;
  reason: string;
  headers?: any;
}) {
  db.query(
    `INSERT INTO unauthorized_logs (ip, reason, headers) VALUES (?, ?, ?)`,
  ).run(ip ?? "unknown", reason, JSON.stringify(headers ?? {}));
}

export const limiter = rateLimit({
  max: 10,
  duration: 60_000,
  headers: true,
});
