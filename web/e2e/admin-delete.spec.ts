import type { Page } from "@playwright/test";

import { env, expect, expectNotFound, test, uniqueSlug } from "./admin";
import { createPost } from "./fixtures";

function published(slug: string) {
  return {
    slug,
    title: `Delete me ${slug}`,
    summary: "Written by an E2E spec.",
    body_md: "A post that gets deleted.",
    published_at: new Date(Date.now() - 60_000).toISOString(),
    tags: [],
  };
}

function deleteDialog(page: Page) {
  return page.getByRole("dialog", { name: "Delete this post?" });
}

test("deleting from the editor asks first, then removes the post", async ({
  page,
  request,
}, testInfo) => {
  const slug = uniqueSlug(testInfo, "delete-editor");
  const post = await createPost(request, env, published(slug));

  // Cache the public page first, so the delete has to expire it.
  await page.goto(`/posts/${slug}`);
  await expect(
    page.getByRole("heading", { level: 1, name: post.title }),
  ).toBeVisible();

  await page.goto(`/admin/posts/${post.id}`);
  await page.getByRole("button", { name: "Delete post" }).click();
  await expect(deleteDialog(page)).toContainText(`“${post.title}”`);

  await deleteDialog(page).getByRole("button", { name: "Cancel" }).click();
  await expect(deleteDialog(page)).toHaveCount(0);
  await expect(page).toHaveURL(new RegExp(`/admin/posts/${post.id}$`));

  await page.getByRole("button", { name: "Delete post" }).click();
  await deleteDialog(page).getByRole("button", { name: "Delete post" }).click();

  await expect(page).toHaveURL(/\/admin$/);
  await expect(page.getByRole("link", { name: post.title })).toHaveCount(0);
  await expectNotFound(page, `/posts/${slug}`);
  await expectNotFound(page, `/admin/posts/${post.id}`);
});

test("deleting from the post list removes the row", async ({
  page,
  request,
}, testInfo) => {
  const slug = uniqueSlug(testInfo, "delete-list");
  const post = await createPost(request, env, published(slug));

  await page.goto("/admin");
  const trash = page.getByRole("button", { name: `Delete ${post.title}` });
  await trash.click();
  await expect(deleteDialog(page)).toContainText(`“${post.title}”`);

  await page.keyboard.press("Escape");
  await expect(deleteDialog(page)).toHaveCount(0);
  await expect(trash).toBeVisible();

  await trash.click();
  await deleteDialog(page).getByRole("button", { name: "Delete post" }).click();

  await expect(deleteDialog(page)).toHaveCount(0);
  await expect(trash).toHaveCount(0);
  await expect(page).toHaveURL(/\/admin$/);
  await expectNotFound(page, `/posts/${slug}`);
});
