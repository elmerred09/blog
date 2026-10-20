// slugify mirrors internal/slug.Make, so the slug suggested from a title is
// the one the API would accept: lowercase ASCII letters and digits, each run
// of anything else one dash, accents folded. It returns "" when nothing
// usable is left; the API then reports the slug as invalid.
export function slugify(title: string): string {
  return title
    .toLowerCase()
    .normalize("NFD")
    .replace(/\p{Mn}/gu, "")
    .replace(/[^a-z0-9]+/g, "-")
    .replace(/^-+|-+$/g, "");
}
