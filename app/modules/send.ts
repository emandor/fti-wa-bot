import { Elysia } from "elysia";
import { SendRequestSchema, SendResponseSchema } from "../../types/send";
import { insertLog } from "../services/log";
import { env } from "../../config/env";
import { logUnauthorizedAttempt } from "../services/security";
import type { WASocket } from "@whiskeysockets/baileys";

export const createSendModule = (sock: WASocket) =>
  new Elysia({ name: "send-module" }).post(
    "/send",
    async ({ body, headers }) => {
      if (headers.authorization !== `Bearer ${env.authToken}`) {
        logUnauthorizedAttempt({
          ip: headers["x-forwarded-for"] ?? "local",
          reason: "Unauthorized",
          headers,
        });

        return {
          success: false,
          sent_to: "unauthorized",
          timestamp: new Date().toISOString(),
        };
      }

      const groupId = body.groupId || env.groupJid;
      await sock.sendMessage(groupId, { text: body.message });
      insertLog({ message: body.message, groupId });
<<<<<<< Updated upstream
=======
      console.log(`[WA] Message sent to ${groupId}: ${body.message}`);
>>>>>>> Stashed changes

      return {
        success: true,
        sent_to: groupId,
        timestamp: new Date().toISOString(),
      };
    },
    {
      body: SendRequestSchema,
      response: SendResponseSchema,
    },
  );
