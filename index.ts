import { Elysia } from "elysia";
import { initWhatsApp } from "./app/services/wa";
import { createSendModule } from "./app/modules/send";
import { limiter } from "./app/services/security";
import { env } from "./config/env";

async function main() {
  const sock = await initWhatsApp();

  new Elysia()
    .use(limiter)
    .use(createSendModule(sock))
    .onStart(() =>
      console.log(`🚀 Server ready at http://localhost:${env.port}`),
    )
    .listen(env.port);
}

main().catch((error) => {
  console.error("Failed to start the server:", error);
  process.exit(1);
});

