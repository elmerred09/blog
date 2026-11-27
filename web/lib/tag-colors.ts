import type { Tag } from "@/lib/api";

export type TagColor = Tag["color"];

// Each palette color's chip classes, written out in full so Tailwind finds
// them in the source. The colors are tokens in app/globals.css. Keep the
// names in step with tags.Colors in Go and tags_color_check.
const chipClasses: Record<TagColor, string> = {
  slate: "bg-tag-slate text-tag-slate-text",
  coral: "bg-tag-coral text-tag-coral-text",
  rust: "bg-tag-rust text-tag-rust-text",
  amber: "bg-tag-amber text-tag-amber-text",
  yellow: "bg-tag-yellow text-tag-yellow-text",
  olive: "bg-tag-olive text-tag-olive-text",
  brown: "bg-tag-brown text-tag-brown-text",
  forest: "bg-tag-forest text-tag-forest-text",
  green: "bg-tag-green text-tag-green-text",
  teal: "bg-tag-teal text-tag-teal-text",
  petrol: "bg-tag-petrol text-tag-petrol-text",
  blue: "bg-tag-blue text-tag-blue-text",
  navy: "bg-tag-navy text-tag-navy-text",
  indigo: "bg-tag-indigo text-tag-indigo-text",
  purple: "bg-tag-purple text-tag-purple-text",
  pink: "bg-tag-pink text-tag-pink-text",
};

// The admin's swatch for each color: the tint, ringed in the text color.
const swatchClasses: Record<TagColor, string> = {
  slate: "bg-tag-slate border-tag-slate-text",
  coral: "bg-tag-coral border-tag-coral-text",
  rust: "bg-tag-rust border-tag-rust-text",
  amber: "bg-tag-amber border-tag-amber-text",
  yellow: "bg-tag-yellow border-tag-yellow-text",
  olive: "bg-tag-olive border-tag-olive-text",
  brown: "bg-tag-brown border-tag-brown-text",
  forest: "bg-tag-forest border-tag-forest-text",
  green: "bg-tag-green border-tag-green-text",
  teal: "bg-tag-teal border-tag-teal-text",
  petrol: "bg-tag-petrol border-tag-petrol-text",
  blue: "bg-tag-blue border-tag-blue-text",
  navy: "bg-tag-navy border-tag-navy-text",
  indigo: "bg-tag-indigo border-tag-indigo-text",
  purple: "bg-tag-purple border-tag-purple-text",
  pink: "bg-tag-pink border-tag-pink-text",
};

export function tagSwatchClasses(color: TagColor): string {
  return swatchClasses[color] ?? swatchClasses.slate;
}

// The palette in the order the admin offers it. slate is the default.
export const tagColors = Object.keys(chipClasses) as TagColor[];

// A color the API adds before this build knows it shows as slate.
export function tagChipClasses(color: TagColor): string {
  return chipClasses[color] ?? chipClasses.slate;
}
