"use server";

import { updateTag } from "next/cache";

import {
  parsePreviewInput,
  parseSavePostInput,
  parseTagColorInput,
  parseUploadInput,
  type SavePostInput,
  type TagColorInput,
} from "@/lib/admin/input";
import { requireAdmin } from "@/lib/admin/session";
import * as api from "@/lib/api/admin";
import type { Outcome } from "@/lib/api/admin";
import type { AdminPost, Tag, Upload, UploadInput } from "@/lib/api/types";

export type { SavePostInput, TagColorInput };

// Each action checks the session itself: an action is reachable by POST from
// anywhere, whatever page it was rendered on.

// savePost creates or updates a post. On success it expires the cached list
// (which search shares) and the post's page under both slugs, so the next
// public request reads the write.
export async function savePost(
  input: SavePostInput,
): Promise<Outcome<AdminPost>> {
  const admin = await requireAdmin();
  const { id, loadedSlug, updatedAt, post } = parseSavePostInput(input);

  const result =
    id === null
      ? await api.createAdminPost(admin, post)
      : await api.updateAdminPost(admin, id, {
          ...post,
          updated_at: updatedAt!,
        });
  if (result.ok) {
    updateTag("posts");
    updateTag(`post:${result.value.slug}`);
    if (loadedSlug !== null && loadedSlug !== result.value.slug) {
      updateTag(`post:${loadedSlug}`);
    }
  }
  return result;
}

export async function previewPost(bodyMd: string): Promise<Outcome<string>> {
  const admin = await requireAdmin();
  return api.previewPost(admin, parsePreviewInput(bodyMd));
}

// The browser PUTs the file to the returned URL itself; the bytes never pass
// through Next.
export async function createImageUpload(
  input: UploadInput,
): Promise<Outcome<Upload>> {
  const admin = await requireAdmin();
  return api.createImageUpload(admin, parseUploadInput(input));
}

// setTagColor changes a tag's chip color. Every cached list and post page
// carries the posts cache tag, so expiring it repaints the chip everywhere.
export async function setTagColor(input: TagColorInput): Promise<Outcome<Tag>> {
  const admin = await requireAdmin();
  const { slug, color } = parseTagColorInput(input);
  const result = await api.setTagColor(admin, slug, color);
  if (result.ok) {
    updateTag("posts");
  }
  return result;
}
