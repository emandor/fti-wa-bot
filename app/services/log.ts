import { Database } from "bun:sqlite";
const db = new Database("logs.db");

db.query(
  `
  CREATE TABLE IF NOT EXISTS logs (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    message TEXT,
    group_id TEXT,
    timestamp DATETIME DEFAULT CURRENT_TIMESTAMP
  )
`,
).run();

export function insertLog({
  message,
  groupId,
}: {
  message: string;
  groupId: string;
}) {
  db.query(`INSERT INTO logs (message, group_id) VALUES (?, ?)`).run(
    message,
    groupId,
  );
}
