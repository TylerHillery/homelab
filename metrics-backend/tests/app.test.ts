import { expect, test } from "vitest";

import app from "../src/app.js";

test("GET /health-check should return status OK", async () => {
  const response = await app.inject({
    method: "GET",
    url: "/health-check",
  });

  expect(response.statusCode).toBe(200);
  expect(response.json()).toEqual({ status: "ok" });
});
