import {
  test as base,
  expect,
  type Page,
  type TestInfo,
} from "@playwright/test";

import { readE2EEnv } from "./env";

export { expect };

export const env = readE2EEnv();

// The admin specs' test: fails on an uncaught error or a hydration warning
// in any of the test's tabs, so the editor's client-only loading stays
// clean. The fixture argument is provide, not use: eslint's hooks rule
// reads use(...) as React's hook.
export const test = base.extend<{ browserErrors: void }>({
  browserErrors: [
    async ({ context }, provide) => {
      const errors: string[] = [];
      context.on("weberror", (error) => {
        errors.push(`uncaught: ${error.error().message}`);
      });
      context.on("console", (message) => {
        if (message.type() === "error" && /hydrat/i.test(message.text())) {
          errors.push(message.text());
        }
      });
      await provide();
      expect(errors, "errors in the browser").toEqual([]);
    },
    { auto: true },
  ],
});

// uniqueSlug keeps posts apart across workers, --repeat-each and retries.
// The specs only ever look up their own posts by these slugs.
export function uniqueSlug(testInfo: TestInfo, label: string): string {
  const run = [
    testInfo.workerIndex,
    testInfo.repeatEachIndex,
    testInfo.retry,
    Date.now().toString(36),
  ].join("-");
  return `e2e-${label}-${run}`;
}

// The editor loads in the browser only, after the page; wait for its body.
export async function openEditor(page: Page, path: string): Promise<void> {
  await page.goto(path);
  await expect(page.getByRole("textbox", { name: "Body" })).toBeVisible();
}

// By role, not label: after the first save, the new-post form stays mounted
// but hidden, and role queries skip hidden elements.
export function field(page: Page, name: "Title" | "Slug" | "Summary") {
  return page.getByRole("textbox", { name, exact: true });
}

export async function fillPost(
  page: Page,
  post: { title: string; slug: string; summary?: string; body?: string },
): Promise<void> {
  await field(page, "Title").fill(post.title);
  await field(page, "Slug").fill(post.slug);
  await field(page, "Summary").fill(post.summary ?? "Written by an E2E spec.");
  if (post.body) {
    await page.getByRole("textbox", { name: "Body" }).click();
    await page.keyboard.type(post.body);
  }
}

type State = "draft" | "published" | "scheduled";

// The Publishing radio a loaded post of each state has checked.
const stateRadio: Record<State, RegExp> = {
  draft: /^Draft$/,
  published: /^Published /,
  scheduled: /^Schedule$/,
};

// save clicks Save and waits for the saved state. A new post's first save
// moves to its edit page, whose fresh editor has no status line, so there it
// checks the Publishing radio the loaded post has.
export async function save(page: Page, state: State): Promise<void> {
  const isNew = new URL(page.url()).pathname === "/admin/posts/new";
  await page.getByRole("button", { name: "Save" }).click();
  if (isNew) {
    await page.waitForURL(/\/admin\/posts\/[0-9a-f-]{36}$/);
    await expect(
      page.getByRole("radio", { name: stateRadio[state] }),
    ).toBeChecked();
    return;
  }
  await expect(page.getByRole("status")).toHaveText(`Saved (${state}).`);
}

export async function expectNotFound(page: Page, path: string): Promise<void> {
  await page.goto(path);
  await expect(
    page.getByRole("heading", { level: 1, name: "Not found" }),
  ).toBeVisible();
}
