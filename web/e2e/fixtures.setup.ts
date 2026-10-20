import { test as setup } from "@playwright/test";

import { readE2EEnv } from "./env";
import { ALL_FIXTURES, createFixture } from "./fixtures";

// Runs once both servers are up and before any spec. Nothing may request a
// public page before this: listPosts and searchPosts are "use cache" with
// the posts tag, posts created through the API don't expire it, so an early
// request would cache a list without the fixtures until next start restarts.
setup("create fixture posts", async ({ request }) => {
  const env = readE2EEnv();
  // One at a time, so created_at follows the list order on every run.
  for (const post of ALL_FIXTURES) {
    await createFixture(request, env, post);
  }
});
