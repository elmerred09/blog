import {
  env,
  expect,
  expectNotFound,
  field,
  fillPost,
  openEditor,
  save,
  test,
  uniqueSlug,
} from "./admin";
import { CODE_POST, createPost } from "./fixtures";

// Every spec writes its own posts under a unique slug and reads only those,
// so they can run in parallel with each other and with the public specs.

test("a draft stays private until it's published, then shows at once", async ({
  page,
  context,
}, testInfo) => {
  const slug = uniqueSlug(testInfo, "publish");
  const title = `Draft then publish ${slug}`;
  await openEditor(page, "/admin/posts/new");
  await fillPost(page, { title, slug, body: "Written as a draft first." });
  await save(page, "draft");

  // Requested while it's a draft, so the not-found page and a list without
  // it are what's cached. Publishing has to expire both.
  const visitor = await context.newPage();
  await expectNotFound(visitor, `/posts/${slug}`);
  await visitor.goto("/");
  await expect(visitor.getByRole("link", { name: title })).toHaveCount(0);

  await page.getByRole("radio", { name: "Publish now" }).check();
  await save(page, "published");

  await visitor.goto(`/posts/${slug}`);
  await expect(
    visitor.getByRole("heading", { level: 1, name: title }),
  ).toBeVisible();
  await expect(visitor.getByText("Written as a draft first.")).toBeVisible();
  await visitor.goto("/");
  await expect(visitor.getByRole("link", { name: title })).toBeVisible();
});

test("renaming a published post's slug moves its page", async ({
  page,
  request,
}, testInfo) => {
  const oldSlug = uniqueSlug(testInfo, "rename-from");
  const newSlug = uniqueSlug(testInfo, "rename-to");
  const title = `Renamed ${oldSlug}`;
  const post = await createPost(request, env, {
    slug: oldSlug,
    title,
    summary: "Renamed by an E2E spec.",
    body_md: "About to move.",
    published_at: new Date(Date.now() - 60_000).toISOString(),
    tags: [],
  });

  // Cached under the old slug before the rename.
  await page.goto(`/posts/${oldSlug}`);
  await expect(
    page.getByRole("heading", { level: 1, name: title }),
  ).toBeVisible();

  await openEditor(page, `/admin/posts/${post.id}`);
  await field(page, "Slug").fill(newSlug);
  await expect(
    page.getByText(
      `This post is published: changing its slug breaks links to /posts/${oldSlug}.`,
    ),
  ).toBeVisible();
  await save(page, "published");

  await expectNotFound(page, `/posts/${oldSlug}`);
  await page.goto(`/posts/${newSlug}`);
  await expect(
    page.getByRole("heading", { level: 1, name: title }),
  ).toBeVisible();
});

test("a slug another post uses is refused beside the field", async ({
  page,
}, testInfo) => {
  await openEditor(page, "/admin/posts/new");
  await fillPost(page, {
    title: `Slug clash ${uniqueSlug(testInfo, "clash")}`,
    slug: CODE_POST.slug,
  });
  await page.getByRole("button", { name: "Save" }).click();

  await expect(
    page.getByText("Another post already uses this slug."),
  ).toBeVisible();
  await expect(field(page, "Slug")).toHaveAttribute("aria-invalid", "true");
  await expect(page).toHaveURL(/\/admin\/posts\/new$/);
});

test("preview renders the body as the post page will", async ({
  page,
}, testInfo) => {
  const slug = uniqueSlug(testInfo, "preview");
  const title = `Preview ${slug}`;
  await openEditor(page, "/admin/posts/new");
  await fillPost(page, { title, slug, body: "Before the code." });
  await page.keyboard.press("Enter");
  await page.getByRole("button", { name: "Code block" }).click();
  await page.keyboard.type("func main() {}");
  await page.getByRole("textbox", { name: "Code language" }).fill("go");

  await page.getByRole("tab", { name: "Preview" }).click();
  const preview = page.getByRole("article");
  await expect(
    preview.getByRole("heading", { level: 1, name: title }),
  ).toBeVisible();
  await expect(preview.getByText("Before the code.")).toBeVisible();
  const code = preview.locator("pre.chroma");
  await expect(code).toContainText("func main() {}");
  await expect(code.locator("span.kd", { hasText: "func" })).toBeVisible();

  // Back to writing: the editor was hidden, not unmounted, so the text stays.
  await page.getByRole("tab", { name: "Write" }).click();
  await expect(page.getByRole("textbox", { name: "Body" })).toContainText(
    "Before the code.",
  );
});

test("tags added and removed in the editor are the ones published", async ({
  page,
}, testInfo) => {
  const slug = uniqueSlug(testInfo, "tags");
  await openEditor(page, "/admin/posts/new");
  await fillPost(page, { title: `Tagged ${slug}`, slug });

  // A combobox, not a textbox: it has a datalist of existing tags.
  const tags = page.getByRole("combobox", { name: "Tags" });
  await tags.fill("Alpha E2E");
  await tags.press("Enter");
  // A comma commits too, typed or pasted.
  await tags.fill("Beta E2E,Gamma E2E,");
  await page.getByRole("button", { name: "Remove Gamma E2E" }).click();
  await expect(page.getByRole("button", { name: /^Remove / })).toHaveCount(2);

  await page.getByRole("radio", { name: "Publish now" }).check();
  await save(page, "published");

  await page.goto(`/posts/${slug}`);
  await expect(
    page.getByRole("list", { name: "Tags" }).getByRole("listitem"),
  ).toHaveText(["Alpha E2E", "Beta E2E"]);
});

test("saving over a newer save shows the conflict", async ({
  page,
  context,
  request,
}, testInfo) => {
  const slug = uniqueSlug(testInfo, "conflict");
  const post = await createPost(request, env, {
    slug,
    title: `Conflict ${slug}`,
    summary: "Edited in two tabs.",
    body_md: "Two tabs.",
    published_at: null,
    tags: [],
  });
  const other = await context.newPage();
  await openEditor(page, `/admin/posts/${post.id}`);
  await openEditor(other, `/admin/posts/${post.id}`);

  await field(page, "Title").fill(`First tab ${slug}`);
  await save(page, "draft");

  await field(other, "Title").fill(`Second tab ${slug}`);
  await other.getByRole("button", { name: "Save" }).click();
  await expect(
    other.getByText(
      "This post changed since you opened it, maybe in another tab. Copy your changes, reload, and apply them again.",
    ),
  ).toBeVisible();
  await expect(other.getByRole("status")).toHaveCount(0);
});
