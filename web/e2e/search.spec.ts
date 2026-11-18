import { expect, test } from "@playwright/test";

import { SEARCH_POST, SEARCH_TERM } from "./fixtures";

// SEARCH_TERM is in the search fixture, the draft and the scheduled post,
// and in nothing the admin specs write, so exactly one result means neither
// unpublished post leaked.

test("searching from the header bar finds the one published match", async ({
  page,
}) => {
  await page.goto("/");
  const searchbox = page.getByRole("searchbox", { name: "Search posts" });
  // The shortcut's listener attaches on hydration; a press before that is lost.
  await expect(async () => {
    await page.keyboard.press("/");
    await expect(searchbox).toBeFocused({ timeout: 500 });
  }).toPass();
  await searchbox.fill(SEARCH_TERM);
  await searchbox.press("Enter");

  await expect(page).toHaveURL(new RegExp(`/search\\?q=${SEARCH_TERM}$`));
  await expect(
    page.getByRole("heading", {
      level: 1,
      name: `Results for “${SEARCH_TERM}”`,
    }),
  ).toBeVisible();
  const results = page.getByRole("article");
  await expect(results).toHaveCount(1);
  const link = results.getByRole("link", { name: SEARCH_POST.title });
  await expect(link).toBeVisible();
  // The snippet marks the matched words.
  await expect(results.locator("mark").first()).toHaveText(
    new RegExp(SEARCH_TERM, "i"),
  );

  await link.click();
  await expect(page).toHaveURL(new RegExp(`/posts/${SEARCH_POST.slug}$`));
  await expect(
    page.getByRole("heading", { level: 1, name: SEARCH_POST.title }),
  ).toBeVisible();
});

test("a search with no matches says so and keeps the query", async ({
  page,
}) => {
  const query = "zyzzyvanomatch";
  await page.goto(`/search?q=${query}`);

  await expect(page.getByText(`No posts match “${query}”.`)).toBeVisible();
  await expect(page.getByRole("article")).toHaveCount(0);
  await expect(
    page.getByRole("searchbox", { name: "Search posts" }),
  ).toHaveValue(query);
});

test("a narrow screen searches from the page's own form", async ({ page }) => {
  await page.setViewportSize({ width: 390, height: 844 });
  await page.goto("/");
  await page.getByRole("link", { name: "Search", exact: true }).click();

  await expect(page).toHaveURL(/\/search$/);
  const searchbox = page.getByRole("searchbox", { name: "Search posts" });
  await searchbox.fill(SEARCH_TERM);
  await page.getByRole("button", { name: "Search" }).click();

  await expect(page).toHaveURL(new RegExp(`/search\\?q=${SEARCH_TERM}$`));
  await expect(page.getByRole("article")).toHaveCount(1);
  await expect(searchbox).toHaveValue(SEARCH_TERM);
});
