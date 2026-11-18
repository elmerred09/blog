import { ogSize, siteImage } from "@/lib/og";
import { siteName } from "@/lib/site";

export const alt = siteName;
export const size = ogSize;
export const contentType = "image/png";

export default function Image() {
  return siteImage();
}
