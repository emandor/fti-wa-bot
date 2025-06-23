import makeWASocket, {
  fetchLatestBaileysVersion,
  makeCacheableSignalKeyStore,
  DisconnectReason,
} from "@whiskeysockets/baileys";
import * as qrcode from "qrcode-terminal";
import { Boom } from "@hapi/boom";
import { createSQLiteAuthState } from "./auth_wa";
import { logger } from "./logger";

export async function initWhatsApp() {
  const { version } = await fetchLatestBaileysVersion();
  const { state, saveCreds } = createSQLiteAuthState();

  const sock = makeWASocket({
    version,
    auth: {
      creds: state.creds,
      keys: makeCacheableSignalKeyStore(state.keys, logger),
    },
    logger,
    printQRInTerminal: false,
  });

  sock.ev.on("connection.update", ({ connection, lastDisconnect, qr }) => {
    if (qr) qrcode.generate(qr, { small: true });

    if (connection === "close") {
      const status = (lastDisconnect?.error as Boom)?.output?.statusCode;
      if (status !== DisconnectReason.loggedOut) {
        console.log("[WA] Connection closed. Reconnecting...");
        initWhatsApp();
      } else {
        console.log("[WA] Logged out. Need to re-scan QR.");
      }
    }

    if (connection === "open") {
      console.log("[WA] Connection established.");
    }
  });

  sock.ev.on("creds.update", saveCreds);
  sock.ev.on("chats.upsert", (c) => {
    c.forEach((chat) => {
      const name = chat.name || "(no name)";
      console.log(`[${chat.id}] => ${name}`);
    });
    // for (const chat of chats) {
    //   const name = chat.name || chat.subject || "(no name)";
    //   console.log(`[${chat.id}] => ${name}`);
    // }
  });
  sock.ev.on("messages.upsert", ({ type, messages }) => {
    if (type == "notify") {
      // new messages
      for (const message of messages) {
        console.log(
          `[${message.key.remoteJid}] => ${message.message?.conversation || "(no message)"}`,
        );
        // messages is an array, do not just handle the first message, you will miss messages
      }
    } else {
      // old already seen / handled messages
      // handle them however you want to
    }
  });

  return sock;
}
