import fastify from "fastify";

const app = fastify({ logger: true });

app.get("/ping", async function handler() {
  return "pong";
});

try {
  await app.listen({ port: 3000 });
} catch {
  process.exit(1);
}
