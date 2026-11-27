import { env, expect, test, uniqueSlug } from "./admin";
import { createPost } from "./fixtures";

// --tag-teal in app/globals.css, as the browser reports it.
const TEAL = "rgb(213, 247, 244)";
const SLATE = "rgb(234, 239, 244)";

test("a tag's color shows on its chip once the admin picks it", async ({
  page,
  request,
}, testInfo) => {
  const slug = uniqueSlug(testInfo, "tag-color");
  const tagName = `Color ${slug}`;
  await createPost(request, env, {
    slug,
    title: `Tag color ${slug}`,
    summary: "Written by an E2E spec.",
    body_md: "A post whose only tag gets a color.",
    published_at: new Date(Date.now() - 60_000).toISOString(),
    tags: [tagName],
  });

  // Load the post page first, so it is cached with the default color and the
  // save has to expire it.
  await page.goto(`/posts/${slug}`);
  const chip = page
    .getByRole("list", { name: "Tags" })
    .getByRole("link", { name: tagName });
  await expect(chip).toHaveCSS("background-color", SLATE);

  await page.goto("/admin/tags");
  await expect(
    page.getByRole("heading", { level: 1, name: "Tags" }),
  ).toBeVisible();
  const row = page.getByRole("listitem").filter({ hasText: tagName });
  const swatches = row.getByRole("group", { name: `Color for ${tagName}` });
  await expect(swatches.getByRole("radio", { name: "Slate" })).toBeChecked();

  await swatches.getByTitle("Teal").click();
  await expect(row.getByRole("status")).toHaveText("Saved");
  await expect(swatches.getByRole("radio", { name: "Teal" })).toBeChecked();

  await page.reload();
  await expect(
    page
      .getByRole("group", { name: `Color for ${tagName}` })
      .getByRole("radio", { name: "Teal" }),
  ).toBeChecked();

  await page.goto(`/posts/${slug}`);
  await expect(chip).toHaveCSS("background-color", TEAL);
});

test("the admin header links the posts and tags pages", async ({ page }) => {
  await page.goto("/admin/tags");
  const nav = page.getByRole("navigation", { name: "Admin" });
  await expect(nav.getByRole("link", { name: "Tags" })).toHaveAttribute(
    "aria-current",
    "page",
  );

  await nav.getByRole("link", { name: "Posts" }).click();
  await expect(page).toHaveURL(/\/admin$/);
  await expect(nav.getByRole("link", { name: "Posts" })).toHaveAttribute(
    "aria-current",
    "page",
  );
});
