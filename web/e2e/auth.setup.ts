import { expect, test as setup } from "@playwright/test";

import {
  ADMIN_NAME,
  ADMIN_STORAGE_STATE,
  VERSION_GUARD_MESSAGE,
  addSessionCookies,
  mintSessionCookies,
} from "./auth";
import { adminGithubId, readE2EEnv } from "./env";

// Signs in once for the admin project: mints a session for the allowed
// GitHub id, checks the editor accepts it, and saves the cookies as the
// storageState every admin spec starts from. Each spec gets its own browser
// context from that file, so a spec that signs out doesn't affect the others.
setup("sign in as the admin", async ({ page, context, baseURL }) => {
  const env = readE2EEnv();
  await addSessionCookies(
    context,
    baseURL!,
    await mintSessionCookies({
      secret: env.web.BETTER_AUTH_SECRET,
      githubId: adminGithubId(env),
      // Longer than any run, shorter than the cookie cache's seven days.
      maxAgeSeconds: 6 * 60 * 60,
    }),
  );
  await page.goto("/admin");
  await expect(page.getByText(ADMIN_NAME), VERSION_GUARD_MESSAGE).toBeVisible();
  await context.storageState({ path: ADMIN_STORAGE_STATE });
});
