import { expect, test } from "@playwright/test";

import { CODE_POST, DRAFT_POST, SCHEDULED_POST } from "./fixtures";

// Indexing is off in this suite, so robots.txt disallows everything. The
// sitemap still lists published posts. Assert on the body, not status codes:
// not-found pages are streamed with a 200.

test("robots.txt disallows the whole site", async ({ page }) => {
  await page.goto("/robots.txt");
  await expect(page.locator("body")).toContainText("Disallow: /");
  await expect(page.locator("body")).not.toContainText("Allow: /");
});

test("the sitemap lists a published fixture and skips drafts", async ({
  page,
}) => {
  await page.goto("/sitemap.xml");
  const body = page.locator("body");
  await expect(body).toContainText(`/posts/${CODE_POST.slug}`);
  await expect(body).not.toContainText(DRAFT_POST.slug);
  await expect(body).not.toContainText(SCHEDULED_POST.slug);
});

test("a post page's Open Graph title matches, and its image is a PNG", async ({
  page,
}) => {
  await page.goto(`/posts/${CODE_POST.slug}`);
  const title = page.locator('meta[property="og:title"]');
  await expect(title).toHaveAttribute("content", CODE_POST.title);

  const image = await page
    .locator('meta[property="og:image"]')
    .getAttribute("content");
  expect(image).toBeTruthy();
  const response = await page.request.get(image!);
  expect(response.headers()["content-type"]).toContain("image/png");
});
