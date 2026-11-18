import { readFile } from "node:fs/promises";
import { join } from "node:path";
import { ImageResponse } from "next/og";
import type { ReactNode } from "react";

import { logoGlyph, logoRadius } from "@/components/logo";
import type { Post } from "@/lib/api";
import { formatDate, formatReadingTime } from "@/lib/format";
import { siteDescription, siteName } from "@/lib/site";

export const ogSize = { width: 1200, height: 630 };

// next/og needs local TTF or OTF files (not variable fonts or WOFF2); reading
// them here means an image never fetches a font. Inter matches the site.
const fontDir = join(process.cwd(), "assets/fonts");
const fonts = await Promise.all(
  (
    [
      ["Inter-Regular.ttf", 400],
      ["Inter-SemiBold.ttf", 600],
      ["Inter-Bold.ttf", 700],
    ] as const
  ).map(async ([file, weight]) => ({
    name: "Inter",
    data: await readFile(join(fontDir, file)),
    style: "normal" as const,
    weight,
  })),
);

// next/og can't read CSS variables; these copy the tokens in globals.css.
const color = {
  foreground: "#000000",
  muted: "rgba(0, 0, 0, 0.6)",
  brand: "#2b6fd6",
  brandStrong: "#1f55a8",
  chip: "#e1ecfb",
};

const maxTags = 3;

// titleSize shrinks long titles so they fit in three lines.
export function titleSize(title: string): number {
  if (title.length <= 40) {
    return 76;
  }
  if (title.length <= 72) {
    return 64;
  }
  return 52;
}

export function postImage(
  post: Pick<Post, "title" | "published_at" | "reading_minutes" | "tags">,
): ImageResponse {
  return card(
    <div
      style={{
        display: "block",
        fontSize: titleSize(post.title),
        fontWeight: 700,
        lineHeight: 1.1,
        letterSpacing: "-0.03em",
        lineClamp: 3,
      }}
    >
      {post.title}
    </div>,
    <div style={{ display: "flex", alignItems: "center", gap: 24 }}>
      <div style={{ display: "flex", fontSize: 28, color: color.muted }}>
        {`${formatDate(post.published_at)}  ·  ${formatReadingTime(post.reading_minutes)}`}
      </div>
      <div style={{ display: "flex", gap: 12 }}>
        {post.tags.slice(0, maxTags).map((tag) => (
          <div
            key={tag.slug}
            style={{
              display: "flex",
              padding: "6px 16px",
              borderRadius: 8,
              background: color.chip,
              color: color.brandStrong,
              fontSize: 24,
              fontWeight: 600,
            }}
          >
            {tag.name}
          </div>
        ))}
      </div>
    </div>,
  );
}

// siteImage repeats the home page's headline, highlight included.
export function siteImage(): ImageResponse {
  return card(
    <div style={{ display: "flex", flexDirection: "column", gap: 32 }}>
      <div
        style={{
          display: "flex",
          flexWrap: "wrap",
          fontSize: 76,
          fontWeight: 700,
          lineHeight: 1.1,
          letterSpacing: "-0.03em",
        }}
      >
        <span>Making sense of software,&nbsp;</span>
        <span style={{ display: "flex", position: "relative" }}>
          <span
            style={{
              position: "absolute",
              left: -10,
              right: -10,
              bottom: 6,
              height: 32,
              borderRadius: "3px 16px 6px 12px",
              background: color.brand,
              opacity: 0.45,
              transform: "rotate(-2deg)",
            }}
          />
          bit by bit
        </span>
        <span>.</span>
      </div>
      <div
        style={{
          display: "flex",
          maxWidth: 900,
          fontSize: 30,
          lineHeight: 1.4,
          color: color.muted,
        }}
      >
        {siteDescription}
      </div>
    </div>,
  );
}

// card is the shared frame: the content centered vertically, a bottom row
// with the header's logo and name (and any detail on the right), and a
// brand-colored edge along the bottom.
function card(main: ReactNode, detail?: ReactNode): ImageResponse {
  return new ImageResponse(
    <div
      style={{
        width: "100%",
        height: "100%",
        display: "flex",
        flexDirection: "column",
        background: "#ffffff",
        color: color.foreground,
        fontFamily: "Inter",
      }}
    >
      <div
        style={{
          flex: 1,
          display: "flex",
          flexDirection: "column",
          padding: "64px 80px 56px",
        }}
      >
        <div style={{ flex: 1, display: "flex", alignItems: "center" }}>
          {main}
        </div>
        <div
          style={{
            display: "flex",
            alignItems: "center",
            justifyContent: "space-between",
          }}
        >
          <div style={{ display: "flex", alignItems: "center", gap: 20 }}>
            <svg width={64} height={64} viewBox="0 0 32 32">
              <rect width="32" height="32" rx={logoRadius} fill={color.brand} />
              <path d={logoGlyph} fill="#ffffff" />
            </svg>
            <div
              style={{
                display: "flex",
                fontSize: 40,
                fontWeight: 700,
                letterSpacing: "-0.02em",
              }}
            >
              {siteName}
            </div>
          </div>
          {detail}
        </div>
      </div>
      <div style={{ display: "flex", height: 16, background: color.brand }} />
    </div>,
    { ...ogSize, fonts },
  );
}
