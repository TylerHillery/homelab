import fastifySSE from "@tylerhillery/fastify-sse";
import Fastify from "fastify";

const app = Fastify({ logger: true });
await app.register(fastifySSE);

app.get("/health-check", async () => {
  return { status: "ok" };
});

app.get("/sse", { sse: true }, async (request, reply) => {
  reply.sse.keepAlive();

  await reply.sse.send({
    id: "123",
    event: "test",
    data: { hello: "world" },
  });

  const interval = setInterval(async () => {
    if (reply.sse.isConnected) {
      await reply.sse.send({
        event: "update",
        data: { timestamp: new Date().toISOString() },
      });
    } else {
      clearInterval(interval);
    }
  }, 2000);

  request.raw.on("close", () => {
    clearInterval(interval);
    console.log("Connection closed");
  });
});

export default app;
