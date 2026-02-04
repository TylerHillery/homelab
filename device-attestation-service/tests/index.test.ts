import { afterAll,beforeAll, describe, expect, it } from "@jest/globals";
import fastify from "fastify";

describe("Difference App", () => {
  let app: ReturnType<typeof fastify>;

  beforeAll(async () => {
    app = fastify({ logger: false });

    app.get("/ping", async function handler() {
      return "pong";
    });

    await app.ready();
  });

  afterAll(async () => {
    await app.close();
  });

  it("GET /ping should return pong", async () => {
    const response = await app.inject({
      method: "GET",
      url: "/ping",
    });

    expect(response.statusCode).toBe(200);
    expect(response.body).toBe("pong");
  });
});