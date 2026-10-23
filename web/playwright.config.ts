import { defineConfig, devices } from "@playwright/test";

import { ADMIN_STORAGE_STATE } from "./e2e/auth";
import { readE2EEnv } from "./e2e/env";

const env = readE2EEnv();
const baseURL = env.web.BETTER_AUTH_URL;
const port = new URL(baseURL).port;
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
      // they do in production. Built here so the build sees the same env as
      // the server. The process is standalone's server.js, which is what the
      // image runs; next start warns when output is standalone. public,
      // .next/static, and the Open Graph fonts are not traced, so copy them
      // the way the image does.
      command: [
        "node_modules/.bin/next build",
        "cp -r public .next/standalone/",
        "cp -r .next/static .next/standalone/.next/",
        "mkdir -p .next/standalone/assets",
        "cp -r assets/fonts .next/standalone/assets/",
        `PORT=${port} node .next/standalone/server.js`,
      ].join(" && "),
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
