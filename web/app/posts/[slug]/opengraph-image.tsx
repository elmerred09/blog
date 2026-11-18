import { notFound } from "next/navigation";

import { getPost } from "@/lib/api";
import { ogSize, postImage } from "@/lib/og";
import { siteName } from "@/lib/site";

export const alt = siteName;
export const size = ogSize;
export const contentType = "image/png";

export default async function Image({
  params,
}: {
  params: Promise<{ slug: string }>;
}) {
  const { slug } = await params;
  const post = await getPost(slug);
  if (post === null) {
    notFound();
  }
  return postImage(post);
}
