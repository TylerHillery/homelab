import Fastify from "fastify";

const app = Fastify();

app.get("/health-check", async () => {
  return { status: "ok" };
});

export default app;
