import { describe, expect, it, vi } from "vitest";

import type { Outcome } from "@/lib/api/admin";
import type { Upload } from "@/lib/api/types";

import { altFromFileName, uploadImage } from "./upload";

const upload: Upload = {
  url: "http://s3.test/blog/images/abc.png?X-Amz-Signature=1",
  headers: { "Content-Type": "image/png", "Content-Length": "3" },
  public_url: "http://s3.test/blog/images/abc.png",
};

function png(): File {
  return new File([new Uint8Array([1, 2, 3])], "diagram.png", {
    type: "image/png",
  });
}

function created(): (input: unknown) => Promise<Outcome<Upload>> {
  return vi.fn(async () => ({ ok: true as const, value: upload }));
}

describe("uploadImage", () => {
  it("asks for an upload with the file's type and size", async () => {
    const createUpload = created();
    const put = vi.fn(async () => new Response(null, { status: 200 }));

    await uploadImage(png(), createUpload, put);

    expect(createUpload).toHaveBeenCalledExactlyOnceWith({
      content_type: "image/png",
      size: 3,
    });
  });

  it("PUTs the file to the signed URL and returns its public URL", async () => {
    const put = vi.fn(async () => new Response(null, { status: 200 }));
    const file = png();

    const result = await uploadImage(file, created(), put);

    expect(result).toEqual({ ok: true, url: upload.public_url });
    expect(put).toHaveBeenCalledExactlyOnceWith(upload.url, {
      method: "PUT",
      // Content-Length is a forbidden header in browsers; fetch sets it from
      // the file, which is the size that was signed.
      headers: { "Content-Type": "image/png" },
      body: file,
    });
  });

  it("reports a 403 from the store as a failed upload", async () => {
    const put = vi.fn(async () => new Response("<Error/>", { status: 403 }));

    const result = await uploadImage(png(), created(), put);

    expect(result).toEqual({
      ok: false,
      message: expect.stringMatching(/didn't match/),
    });
  });

  it("reports other store errors with their status", async () => {
    const put = vi.fn(async () => new Response(null, { status: 503 }));

    const result = await uploadImage(png(), created(), put);

    expect(result).toEqual({
      ok: false,
      message: expect.stringContaining("503"),
    });
  });

  it("reports an unreachable store", async () => {
    const put = vi.fn(async () => {
      throw new TypeError("Failed to fetch");
    });

    const result = await uploadImage(png(), created(), put);

    expect(result.ok).toBe(false);
  });

  it("returns the API's problems for a rejected file", async () => {
    const createUpload = vi.fn(async () => ({
      ok: false as const,
      error: {
        code: "invalid_request" as const,
        message: "content_type: not an allowed image type",
        problems: [
          { field: "content_type", message: "not an allowed image type" },
        ],
      },
    }));
    const put = vi.fn();

    const result = await uploadImage(png(), createUpload, put);

    expect(result).toEqual({
      ok: false,
      message: "not an allowed image type",
    });
    expect(put).not.toHaveBeenCalled();
  });

  it("refuses a file with no type without asking the API", async () => {
    const createUpload = created();

    const result = await uploadImage(
      new File(["x"], "x"),
      createUpload,
      vi.fn(),
    );

    expect(result.ok).toBe(false);
    expect(createUpload).not.toHaveBeenCalled();
  });
});

describe("altFromFileName", () => {
  it.each([
    ["query-plan.png", "query plan"],
    ["Screenshot_2026-10-18.webp", "Screenshot 2026 10 18"],
    ["image", "image"],
  ])("turns %j into %j", (name, want) => {
    expect(altFromFileName(name)).toBe(want);
  });
});
