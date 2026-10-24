import { readFile } from "node:fs/promises";
import { join } from "node:path";
import { ImageResponse } from "next/og";

import { siteName } from "@/lib/site";

export const ogSize = { width: 1200, height: 630 };

const fontDir = join(process.cwd(), "assets/fonts");

const fonts = [
  {
    name: "Geist",
    data: await readFile(join(fontDir, "Geist-Regular.ttf")),
    style: "normal" as const,
    weight: 400 as const,
  },
  {
    name: "Geist",
    data: await readFile(join(fontDir, "Geist-SemiBold.ttf")),
    style: "normal" as const,
    weight: 600 as const,
  },
];

// ogImage draws one card. next/og needs a local TTF; the files under
// assets/fonts are read here so the image never fetches a font.
export function ogImage(title: string, detail: string): ImageResponse {
  return new ImageResponse(
    <div
      style={{
        width: "100%",
        height: "100%",
        display: "flex",
        flexDirection: "column",
        justifyContent: "space-between",
        background: "#fafafa",
        color: "#171717",
        padding: "80px",
        fontFamily: "Geist",
      }}
    >
      <div style={{ display: "flex", fontSize: 32, color: "#52525b" }}>
        {siteName}
      </div>
      <div
        style={{
          display: "flex",
          flexDirection: "column",
          width: "1040px",
        }}
      >
        <div
          style={{
            display: "flex",
            fontSize: 72,
            fontWeight: 600,
            lineHeight: 1.15,
            letterSpacing: "-0.03em",
          }}
        >
          {title}
        </div>
        <div
          style={{
            display: "flex",
            marginTop: 28,
            fontSize: 32,
            color: "#52525b",
          }}
        >
          {detail}
        </div>
      </div>
    </div>,
    { ...ogSize, fonts },
  );
}
