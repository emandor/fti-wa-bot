import { BufferJSON, initAuthCreds } from "@whiskeysockets/baileys";
import type {
  AuthenticationCreds,
  AuthenticationState,
  SignalDataSet,
  SignalDataTypeMap,
  SignalKeyStore,
} from "@whiskeysockets/baileys";
import { Database } from "bun:sqlite";

const db = new Database("auth.db");

db.query(
  `
  CREATE TABLE IF NOT EXISTS auth_state (
    id INTEGER PRIMARY KEY,
    creds TEXT,
    keys TEXT
  )
`,
).run();

function loadAuth() {
  const row = db
    .query(`SELECT creds, keys FROM auth_state WHERE id = 1`)
    .get() as { creds: string; keys: string } | undefined;

  if (!row) return { creds: initAuthCreds(), keys: {} };

  return {
    creds: JSON.parse(row.creds, BufferJSON.reviver),
    keys: JSON.parse(row.keys, BufferJSON.reviver),
  };
}

function saveAuth(
  creds: AuthenticationCreds,
  keys: Record<string, Record<string, unknown>>,
) {
  const insert = db.query(`
    INSERT INTO auth_state (id, creds, keys) VALUES (1, ?, ?)
    ON CONFLICT(id) DO UPDATE SET creds = excluded.creds, keys = excluded.keys
  `);
  insert.run(
    JSON.stringify(creds, BufferJSON.replacer),
    JSON.stringify(keys, BufferJSON.replacer),
  );
}

export function createSQLiteAuthState(): {
  state: AuthenticationState;
  saveCreds: () => void;
} {
  const { creds, keys } = loadAuth();

  const keyStore: SignalKeyStore = {
    get: async <T extends keyof SignalDataTypeMap>(type: T, ids: string[]) => {
      const result: { [id: string]: SignalDataTypeMap[T] } = {};
      for (const id of ids) {
        const entry = (keys[type] ?? {})[id];
        if (entry) result[id] = entry as SignalDataTypeMap[T];
      }
      return result;
    },
    set: async (data: SignalDataSet) => {
      for (const category of Object.keys(data) as (keyof SignalDataSet)[]) {
        keys[category] = keys[category] || {};
        Object.assign(keys[category], data[category]);
      }
    },
  };

  return {
    state: {
      creds,
      keys: keyStore,
    },
    saveCreds: () => saveAuth(creds, keys),
  };
}
