import type { NextConfig } from "next";

const nextConfig: NextConfig = {
  // cacheComponents and partialPrefetching (which requires it) are off for
  // now: caching follows the fetch-level model, next: { revalidate, tags } on
  // each API call. See the frontend plan before turning them on.
  turbopack: {
    rules: {
      "*.css": {
        loaders: ["@tailwindcss/turbopack"],
        as: "*.css",
      },
    },
  },
};

export default nextConfig;
