import { expect, test, type Page } from "@playwright/test";

import {
  CODE_POST,
  DRAFT_POST,
  FIXTURE_TAG,
  PUBLISHED_FIXTURES,
  SCHEDULED_POST,
} from "./fixtures";

// The admin specs publish posts while these run, and those are newer than
// every fixture. So these assert on fixture posts and their relative order,
// never on counts or positions. Not-found is streamed with a 200, so it's
// asserted by its heading, not a status code.

// The most pages the list walk follows before giving up on the last one.
const MAX_PAGES = 10;

async function listedTitles(page: Page): Promise<string[]> {
  await expect(page.getByRole("article").first()).toBeVisible();
  return page
    .getByRole("article")
    .getByRole("heading", { level: 2 })
    .allTextContents();
}

test("the posts list pages through every published fixture, newest first", async ({
  page,
}) => {
  await page.goto("/");
  const titles = await listedTitles(page);
  let pages = 1;
  const older = page.getByRole("link", { name: "Older posts" });
  while ((await older.count()) > 0) {
    expect(pages, "too many pages; is the cursor advancing?").toBeLessThan(
      MAX_PAGES,
    );
    await page.goto((await older.getAttribute("href"))!);
    titles.push(...(await listedTitles(page)));
    pages++;
  }

  expect(pages).toBeGreaterThan(1);
  const fixtureTitles = PUBLISHED_FIXTURES.map((post) => post.title);
  expect(titles.filter((title) => fixtureTitles.includes(title))).toEqual(
    fixtureTitles,
  );
  expect(titles).not.toContain(DRAFT_POST.title);
  expect(titles).not.toContain(SCHEDULED_POST.title);
});

const FIXTURE_TITLES = new Set(PUBLISHED_FIXTURES.map((post) => post.title));

// fixtureTitlesOn is the fixture posts on the current page, in list order.
// Posts the admin specs publish are newer than every fixture, so they only
// ever land on the first page and are left out of the comparison.
async function fixtureTitlesOn(page: Page): Promise<string[]> {
  return (await listedTitles(page)).filter((title) =>
    FIXTURE_TITLES.has(title),
  );
}

test("Newer posts walks back to the first page", async ({ page }) => {
  await page.goto("/");
  await expect(page.getByRole("link", { name: "Newer posts" })).toHaveCount(0);

  const forward: string[][] = [];
  for (let n = 1; ; n++) {
    forward.push(await fixtureTitlesOn(page));
    const older = page.getByRole("link", { name: "Older posts" });
    if ((await older.count()) === 0) {
      break;
    }
    expect(n, "too many pages; is the cursor advancing?").toBeLessThan(
      MAX_PAGES,
    );
    await page.goto((await older.getAttribute("href"))!);
  }
  expect(forward.length).toBeGreaterThan(1);
  await expect(page.getByRole("link", { name: "Newer posts" })).toBeVisible();

  const back: string[][] = [];
  for (let n = 1; ; n++) {
    const newer = page.getByRole("link", { name: "Newer posts" });
    if ((await newer.count()) === 0) {
      break;
    }
    expect(n, "too many pages; is the cursor advancing?").toBeLessThan(
      MAX_PAGES,
    );
    await page.goto((await newer.getAttribute("href"))!);
    back.push(await fixtureTitlesOn(page));
  }

  // Every page but the first and the last is a fixed keyset slice. The first
  // page can gain posts published while this ran, which only pushes fixtures
  // down, so the returned first page's fixtures are a prefix of the original.
  const stable = forward.slice(1, -1).toReversed();
  expect(back.slice(0, stable.length)).toEqual(stable);
  const returned = back[back.length - 1];
  expect(forward[0].slice(0, returned.length)).toEqual(returned);
});

test("a post page shows its metadata and highlighted code", async ({
  page,
}) => {
  await page.goto(`/posts/${CODE_POST.slug}`);

  await expect(
    page.getByRole("heading", { level: 1, name: CODE_POST.title }),
  ).toBeVisible();
  await expect(page).toHaveTitle(`${CODE_POST.title} | Blog`);
  await expect(page.locator("article time")).toHaveText("Jan 14, 2020");
  await expect(
    page.getByRole("list", { name: "Tags" }).getByRole("listitem"),
  ).toHaveText([FIXTURE_TAG, "Go"]);

  const code = page.locator("pre.chroma");
  await expect(code).toContainText('fmt.Println("hello, e2e")');
  // The HTML only carries chroma's class names; app/chroma.css colors them.
  const keyword = code.locator("span.kd", { hasText: "func" });
  await expect(keyword).toBeVisible();
  const color = (el: Element) => getComputedStyle(el).color;
  expect(await keyword.evaluate(color)).not.toBe(await code.evaluate(color));
});

test("a title in the posts list opens the post", async ({ page }) => {
  await page.goto("/");
  // Whichever post is newest: an admin spec may have published one.
  const [title] = await listedTitles(page);
  await page
    .getByRole("article")
    .getByRole("link", { name: title, exact: true })
    .first()
    .click();
  await expect(page).toHaveURL(/\/posts\/[^/]+$/);
  await expect(
    page.getByRole("heading", { level: 1, name: title, exact: true }),
  ).toBeVisible();
});

const notFoundCases = [
  { name: "an unknown slug", path: "/posts/e2e-no-such-post" },
  { name: "a draft", path: `/posts/${DRAFT_POST.slug}` },
  { name: "a scheduled post", path: `/posts/${SCHEDULED_POST.slug}` },
  { name: "a malformed cursor", path: "/?cursor=not-a-cursor" },
];

for (const { name, path } of notFoundCases) {
  test(`${name} shows not found`, async ({ page }) => {
    await page.goto(path);
    await expect(
      page.getByRole("heading", { level: 1, name: "Not found" }),
    ).toBeVisible();
    await expect(page.getByRole("article")).toHaveCount(0);
  });
}
