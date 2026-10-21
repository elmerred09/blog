import { existsSync, readFileSync } from "node:fs";
import path from "node:path";
import { parseEnv } from "node:util";

const WEB_DIR = path.resolve(__dirname, "..");
const ENV_FILE = path.join(WEB_DIR, "e2e", ".env.e2e");
const DEV_ENV_FILE = path.join(WEB_DIR, ".env.local");

const API_KEYS = [
  "DATABASE_URL",
  "HTTP_ADDR",
  "ADMIN_JWT_PUBLIC_KEY",
  "ADMIN_SUBJECTS",
  "S3_ENDPOINT",
  "S3_REGION",
  "S3_BUCKET",
  "S3_ACCESS_KEY",
  "S3_SECRET_KEY",
  "S3_PUBLIC_BASE_URL",
] as const;

const WEB_KEYS = [
  "API_URL",
  "ADMIN_GITHUB_IDS",
  "ADMIN_JWT_PRIVATE_KEY",
  "BETTER_AUTH_SECRET",
  "BETTER_AUTH_URL",
  "GITHUB_CLIENT_ID",
  "GITHUB_CLIENT_SECRET",
] as const;

type ApiKey = (typeof API_KEYS)[number];
type WebKey = (typeof WEB_KEYS)[number];

export type E2EEnv = {
  api: Record<ApiKey, string>;
  web: Record<WebKey, string>;
};

// readE2EEnv loads web/e2e/.env.e2e, written by scripts/e2e-env.sh. The
// Playwright config passes api and web to the two servers, and the setup
// projects use the same values to sign tokens and mint cookies.
export function readE2EEnv(): E2EEnv {
  if (!existsSync(ENV_FILE)) {
    throw new Error(
      `${path.relative(WEB_DIR, ENV_FILE)} is missing; run make e2e, or scripts/e2e-env.sh <database url>`,
    );
  }
  const vars = parseEnv(readFileSync(ENV_FILE, "utf8"));
  const env = {
    api: pick(vars, API_KEYS),
    web: pick(vars, WEB_KEYS),
  };
  assertShadowsDevEnv(env.web);
  return env;
}

// adminGithubId is the allowed GitHub id the suite signs in and signs API
// tokens as.
export function adminGithubId(env: E2EEnv): string {
  return env.web.ADMIN_GITHUB_IDS.split(",")[0].trim();
}

function pick<K extends string>(
  vars: Record<string, string | undefined>,
  keys: readonly K[],
): Record<K, string> {
  const missing = keys.filter((key) => !vars[key]);
  if (missing.length > 0) {
    throw new Error(
      `${path.basename(ENV_FILE)} is missing ${missing.join(", ")}; rerun scripts/e2e-env.sh`,
    );
  }
  return Object.fromEntries(keys.map((key) => [key, vars[key]!])) as Record<
    K,
    string
  >;
}

// Next loads web/.env.local under next build and next start as well, and
// only process env wins over it, so any key it sets that the suite doesn't
// would reach the test server with the developer's real value.
function assertShadowsDevEnv(web: Record<WebKey, string>) {
  if (!existsSync(DEV_ENV_FILE)) {
    return;
  }
  const devKeys = Object.keys(parseEnv(readFileSync(DEV_ENV_FILE, "utf8")));
  const unshadowed = devKeys.filter((key) => !(key in web));
  if (unshadowed.length > 0) {
    throw new Error(
      `web/.env.local sets ${unshadowed.join(", ")}, which the E2E env doesn't override; add them to scripts/e2e-env.sh and e2e/env.ts`,
    );
  }
}
