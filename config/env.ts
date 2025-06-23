export const env = {
  port: Number(Bun.env.PORT || 5000),
  groupJid: Bun.env.GROUP_JID || "",
  authToken: Bun.env.AUTH_TOKEN || "",
};
