import { t } from "elysia";

export const SendRequestSchema = t.Object({
  message: t.String(),
  groupId: t.Optional(t.String()),
});

export const SendResponseSchema = t.Object({
  success: t.Boolean(),
  sent_to: t.String(),
  timestamp: t.String(),
});
