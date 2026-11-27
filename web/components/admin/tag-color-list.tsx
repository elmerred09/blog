"use client";

import { useRef, useState } from "react";

import { setTagColor } from "@/app/admin/actions";
import type { Tag } from "@/lib/api";
import {
  tagChipClasses,
  tagColors,
  tagSwatchClasses,
  type TagColor,
} from "@/lib/tag-colors";

type Status =
  { kind: "saving" } | { kind: "saved" } | { kind: "error"; message: string };

function label(color: TagColor): string {
  return color[0].toUpperCase() + color.slice(1);
}

// Every tag with a row of palette swatches. Choosing one repaints the chip at
// once and saves in the background; a failed save puts the saved color back
// and says why.
export function TagColorList({ tags }: { tags: Tag[] }) {
  const [shown, setShown] = useState<Record<string, TagColor>>(() =>
    Object.fromEntries(tags.map((t) => [t.slug, t.color])),
  );
  const saved = useRef<Record<string, TagColor>>({ ...shown });
  // The latest pick per tag. Next runs server actions one at a time, so saves
  // land in click order, but only the latest pick's answer may change what
  // the row shows: an earlier save failing must not undo a later pick.
  const latest = useRef<Record<string, number>>({});
  const [status, setStatus] = useState<Record<string, Status>>({});

  if (tags.length === 0) {
    return (
      <p className="text-sm text-muted">
        No tags yet. Tags appear here once a post uses them.
      </p>
    );
  }

  async function choose(slug: string, color: TagColor) {
    const pick = (latest.current[slug] ?? 0) + 1;
    latest.current[slug] = pick;
    setShown((s) => ({ ...s, [slug]: color }));
    setStatus((s) => ({ ...s, [slug]: { kind: "saving" } }));
    let next: Status;
    try {
      const result = await setTagColor({ slug, color });
      if (result.ok) {
        // Every save that lands is the stored color, latest pick or not.
        saved.current[slug] = result.value.color;
        next = { kind: "saved" };
      } else {
        next = { kind: "error", message: result.error.message };
      }
    } catch {
      next = { kind: "error", message: "Couldn't save the color. Try again." };
    }
    if (latest.current[slug] !== pick) {
      return;
    }
    if (next.kind === "error") {
      setShown((s) => ({ ...s, [slug]: saved.current[slug] }));
    }
    setStatus((s) => ({ ...s, [slug]: next }));
  }

  return (
    <ul aria-label="Tags" className="divide-y divide-line">
      {tags.map((tag) => {
        const color = shown[tag.slug];
        const st = status[tag.slug];
        return (
          <li
            key={tag.slug}
            className="flex flex-wrap items-center justify-between gap-x-6 gap-y-3 py-4"
          >
            <div className="flex items-center gap-3">
              <span
                className={`rounded-md px-2.5 py-1 text-xs font-medium ${tagChipClasses(color)}`}
              >
                {tag.name}
              </span>
              {st?.kind === "error" ? (
                <p role="alert" className="text-sm text-red-700">
                  {st.message}
                </p>
              ) : (
                <p role="status" className="text-sm text-muted">
                  {st?.kind === "saving" ? "Saving…" : null}
                  {st?.kind === "saved" ? "Saved" : null}
                </p>
              )}
            </div>
            <fieldset className="flex gap-2">
              <legend className="sr-only">Color for {tag.name}</legend>
              {tagColors.map((c) => (
                <label key={c} className="cursor-pointer" title={label(c)}>
                  <input
                    type="radio"
                    name={`color-${tag.slug}`}
                    value={c}
                    aria-label={label(c)}
                    checked={color === c}
                    onChange={() => choose(tag.slug, c)}
                    className="peer sr-only"
                  />
                  <span
                    aria-hidden="true"
                    className={`block size-7 rounded-full border-2 transition-shadow peer-checked:ring-2 peer-checked:ring-foreground peer-checked:ring-offset-2 peer-focus-visible:outline-2 peer-focus-visible:outline-offset-4 peer-focus-visible:outline-brand ${tagSwatchClasses(c)}`}
                  />
                </label>
              ))}
            </fieldset>
          </li>
        );
      })}
    </ul>
  );
}
