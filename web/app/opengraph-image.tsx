import { ogImage, ogSize } from "@/lib/og";

export const alt = "Blog";
export const size = ogSize;
export const contentType = "image/png";

export default function Image() {
  return ogImage("Blog", "A technical blog about software development.");
}
