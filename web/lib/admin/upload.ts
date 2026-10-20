import type { Outcome } from "@/lib/api/admin";
import type { Upload, UploadInput } from "@/lib/api/types";

export type UploadResult =
  { ok: true; url: string } | { ok: false; message: string };

type CreateUpload = (input: UploadInput) => Promise<Outcome<Upload>>;
type Put = (url: string, init: RequestInit) => Promise<Response>;

// uploadImage asks the API for a presigned PUT, then sends the file straight
// to the store. The signature covers the type and exact size the API was
// told, so the store answers 403 to anything else. Every failure is a value:
// a failed upload shouldn't take the editor down with it.
export async function uploadImage(
  file: File,
  createUpload: CreateUpload,
  put: Put = fetch,
): Promise<UploadResult> {
  if (!file.type) {
    return { ok: false, message: `${file.name} isn't a recognized image.` };
  }

  const created = await createUpload({
    content_type: file.type,
    size: file.size,
  });
  if (!created.ok) {
    const { message, problems } = created.error;
    return {
      ok: false,
      message:
        problems.length > 0
          ? problems.map((p) => p.message).join("; ")
          : message,
    };
  }

  const { url, headers, public_url } = created.value;
  let response: Response;
  try {
    // Only Content-Type: browsers refuse to let scripts set Content-Length,
    // and fetch sends the file's own size, which is what was signed.
    response = await put(url, {
      method: "PUT",
      headers: { "Content-Type": headers["Content-Type"] },
      body: file,
    });
  } catch {
    return { ok: false, message: "Couldn't reach the image store." };
  }

  if (response.status === 403) {
    return {
      ok: false,
      message:
        "The image store refused the file: it didn't match the signed upload.",
    };
  }
  if (!response.ok) {
    return {
      ok: false,
      message: `The image store answered ${response.status}.`,
    };
  }
  return { ok: true, url: public_url };
}

// altFromFileName is a starting alt text: the name without its extension,
// with dashes and underscores as spaces.
export function altFromFileName(name: string): string {
  return name
    .replace(/\.[^.]+$/, "")
    .replace(/[-_]+/g, " ")
    .trim();
}
