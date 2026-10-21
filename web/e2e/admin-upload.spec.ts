import {
  env,
  expect,
  fillPost,
  openEditor,
  save,
  test,
  uniqueSlug,
} from "./admin";

// A 1x1 transparent PNG.
const PNG = Buffer.from(
  "iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVR42mNkYAAAAAYAAjCB0C8AAAAASUVORK5CYII=",
  "base64",
);

test("an uploaded image goes to the store and into the published post", async ({
  page,
  request,
}, testInfo) => {
  const slug = uniqueSlug(testInfo, "upload");
  await openEditor(page, "/admin/posts/new");
  await fillPost(page, { title: `Upload ${slug}`, slug, body: "An image:" });

  // The browser PUTs the file to RustFS itself, from this origin, so this
  // also checks the bucket's CORS rule allows the suite's port.
  const chooser = page.waitForEvent("filechooser");
  await page.getByRole("button", { name: "Image", exact: true }).click();
  await (
    await chooser
  ).setFiles({ name: "e2e-pixel.png", mimeType: "image/png", buffer: PNG });

  const body = page.getByRole("textbox", { name: "Body" });
  const image = body.locator(
    `img[src^="${env.api.S3_PUBLIC_BASE_URL}/images/"]`,
  );
  await expect(image).toHaveAttribute("alt", "e2e pixel");
  await expect(page.getByText(/^Upload failed/)).toHaveCount(0);
  const src = (await image.getAttribute("src"))!;

  await page.getByRole("radio", { name: "Publish now" }).check();
  await save(page, "published");

  await page.goto(`/posts/${slug}`);
  const published = page.getByRole("article").locator(`img[src="${src}"]`);
  await expect(published).toHaveAttribute("alt", "e2e pixel");
  await expect
    .poll(() => published.evaluate((img: HTMLImageElement) => img.naturalWidth))
    .toBe(1);
  const stored = await request.get(src);
  expect(stored.status()).toBe(200);
  expect(stored.headers()["content-type"]).toBe("image/png");
});
