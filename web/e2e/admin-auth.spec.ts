import { expect, test } from "@playwright/test";

import {
  ADMIN_NAME,
  SESSION_DATA_COOKIE,
  SESSION_TOKEN_COOKIE,
  VERSION_GUARD_MESSAGE,
  addSessionCookies,
  mintSessionCookies,
} from "./auth";
import { adminGithubId, readE2EEnv } from "./env";

// Redirects from requireAdmin are streamed (the response is a 200), so these
// assert on where the page ends up, never on a status code.

test("a minted admin session opens the editor", async ({ page }) => {
  await page.goto("/admin");
  await expect(page.getByText(ADMIN_NAME), VERSION_GUARD_MESSAGE).toBeVisible();
  // The post list comes from the admin API, so this also checks the API
  // accepted the token Next signed for this admin.
  await expect(page.getByRole("table")).toBeVisible();
});

test("signing out ends the session", async ({ page, context }) => {
  await page.goto("/admin");
  await expect(page.getByText(ADMIN_NAME)).toBeVisible();

  await page.getByRole("button", { name: "Sign out" }).click();
  await expect(page).toHaveURL(/\/sign-in$/);
  const names = (await context.cookies()).map((cookie) => cookie.name);
  expect(names).not.toContain(SESSION_TOKEN_COOKIE);
  expect(names).not.toContain(SESSION_DATA_COOKIE);

  await page.goto("/admin");
  await expect(page).toHaveURL(/\/sign-in$/);
});

test.describe("without a valid admin session", () => {
  test.use({ storageState: { cookies: [], origins: [] } });

  for (const path of ["/admin", "/admin/posts/new"]) {
    test(`no session cookie: ${path} redirects to sign-in`, async ({
      page,
    }) => {
      await page.goto(path);
      await expect(page).toHaveURL(/\/sign-in$/);
      await expect(
        page.getByRole("heading", { name: "Sign in" }),
      ).toBeVisible();
    });
  }

  test("a GitHub id off the allowlist is turned away", async ({
    page,
    context,
    baseURL,
  }) => {
    const env = readE2EEnv();
    const outsider = String(Number(adminGithubId(env)) + 1);
    await addSessionCookies(
      context,
      baseURL!,
      await mintSessionCookies({
        secret: env.web.BETTER_AUTH_SECRET,
        githubId: outsider,
      }),
    );

    await page.goto("/admin");
    await expect(page).toHaveURL(/\/sign-in\?error=not_allowed$/);
    // By text: Next's route announcer is a role="alert" too.
    await expect(
      page.getByText("That GitHub account isn't allowed to sign in here."),
    ).toBeVisible();
    await expect(page.getByText(ADMIN_NAME)).toBeHidden();
  });

  test("a tampered session cookie is ignored", async ({
    page,
    context,
    baseURL,
  }) => {
    const env = readE2EEnv();
    const cookies = await mintSessionCookies({
      secret: env.web.BETTER_AUTH_SECRET,
      githubId: adminGithubId(env),
    });
    const tampered = cookies.map((cookie) =>
      cookie.name === SESSION_DATA_COOKIE
        ? { ...cookie, value: flipCiphertext(cookie.value) }
        : cookie,
    );
    await addSessionCookies(context, baseURL!, tampered);

    await page.goto("/admin");
    await expect(page).toHaveURL(/\/sign-in$/);
    await expect(page.getByText(ADMIN_NAME)).toBeHidden();
  });
});

// flipCiphertext changes the first character of a compact JWE's ciphertext,
// so decryption fails its authentication tag.
function flipCiphertext(jwe: string): string {
  const parts = jwe.split(".");
  const ciphertext = parts[3];
  parts[3] = (ciphertext[0] === "A" ? "B" : "A") + ciphertext.slice(1);
  return parts.join(".");
}
