import { defineConfig, devices } from "@playwright/test";

import { ADMIN_STORAGE_STATE } from "./e2e/auth";
import { readE2EEnv } from "./e2e/env";

const env = readE2EEnv();
const baseURL = env.web.BETTER_AUTH_URL;
const ci = !!process.env.CI;

export default defineConfig({
  testDir: "./e2e",
  fullyParallel: true,
  forbidOnly: ci,
  retries: ci ? 1 : 0,
  reporter: [[ci ? "github" : "list"], ["html", { open: "never" }]],
  use: {
    baseURL,
    trace: "retain-on-failure",
  },
  projects: [
    { name: "fixtures", testMatch: /fixtures\.setup\.ts$/ },
    // After fixtures, so no browser request reaches the app before the
    // posts exist.
    {
      name: "auth",
      testMatch: /auth\.setup\.ts$/,
      use: { ...devices["Desktop Chrome"] },
      dependencies: ["fixtures"],
    },
    {
      name: "public",
      testMatch: /(public|search|seo)\.spec\.ts$/,
      use: { ...devices["Desktop Chrome"] },
      dependencies: ["fixtures"],
    },
    {
      name: "admin",
      testMatch: /admin-.*\.spec\.ts$/,
      use: { ...devices["Desktop Chrome"], storageState: ADMIN_STORAGE_STATE },
      dependencies: ["auth"],
    },
  ],
  webServer: [
    {
      name: "api",
      command: "go run ./cmd/api",
      cwd: "..",
      env: env.api,
      url: `${env.web.API_URL}/readyz`,
      // go run compiles first, which is slow on a cold CI cache.
      timeout: 180_000,
      stdout: "pipe",
      gracefulShutdown: { signal: "SIGTERM", timeout: 5_000 },
    },
    {
      name: "web",
      // A production build, because "use cache" and updateTag only behave as
      // they do in production under next start. Built here so the build sees
      // the same env as the server. next directly, not npx: killing npx left
      // next-server holding the port.
      command: `node_modules/.bin/next build && node_modules/.bin/next start -p ${new URL(baseURL).port}`,
      env: env.web,
      // Not / or /search: requesting either before the fixtures project runs
      // would cache an empty post list.
      url: `${baseURL}/sign-in`,
      timeout: 300_000,
      stdout: "pipe",
      gracefulShutdown: { signal: "SIGTERM", timeout: 5_000 },
    },
  ],
});
