import type { Tag } from "@/lib/api";

export type TagColor = Tag["color"];

// Each palette color's chip classes, written out in full so Tailwind finds
// them in the source. The colors are tokens in app/globals.css. Keep the
// names in step with tags.Colors in Go and tags_color_check.
const chipClasses: Record<TagColor, string> = {
  slate: "bg-tag-slate text-tag-slate-text",
  rose: "bg-tag-rose text-tag-rose-text",
  orange: "bg-tag-orange text-tag-orange-text",
  amber: "bg-tag-amber text-tag-amber-text",
  green: "bg-tag-green text-tag-green-text",
  teal: "bg-tag-teal text-tag-teal-text",
  violet: "bg-tag-violet text-tag-violet-text",
  pink: "bg-tag-pink text-tag-pink-text",
};

// The admin's swatch for each color: the tint, ringed in the text color.
const swatchClasses: Record<TagColor, string> = {
  slate: "bg-tag-slate border-tag-slate-text",
  rose: "bg-tag-rose border-tag-rose-text",
  orange: "bg-tag-orange border-tag-orange-text",
  amber: "bg-tag-amber border-tag-amber-text",
  green: "bg-tag-green border-tag-green-text",
  teal: "bg-tag-teal border-tag-teal-text",
  violet: "bg-tag-violet border-tag-violet-text",
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
