import { expect, test } from "@playwright/test";

import { SEARCH_POST, SEARCH_TERM } from "./fixtures";

// SEARCH_TERM is in the search fixture, the draft and the scheduled post,
// and in nothing the admin specs write, so exactly one result means neither
// unpublished post leaked.

test("searching from the form finds the one published match", async ({
  page,
}) => {
  await page.goto("/search");
  await page.getByRole("searchbox", { name: "Search posts" }).fill(SEARCH_TERM);
  await page.getByRole("button", { name: "Search" }).click();

  await expect(page).toHaveURL(new RegExp(`/search\\?q=${SEARCH_TERM}$`));
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
