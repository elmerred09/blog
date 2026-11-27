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

// The palette in the order the admin offers it. slate is the default.
export const tagColors = Object.keys(chipClasses) as TagColor[];

// A color the API adds before this build knows it shows as slate.
export function tagChipClasses(color: TagColor): string {
  return chipClasses[color] ?? chipClasses.slate;
}
