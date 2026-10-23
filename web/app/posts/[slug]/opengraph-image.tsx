import { notFound } from "next/navigation";

import { getPost } from "@/lib/api";
import { formatDate } from "@/lib/format";
import { ogImage, ogSize } from "@/lib/og";

export const alt = "Blog post";
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
  return ogImage(post.title, formatDate(post.published_at));
}
