import type { NextConfig } from "next";

const nextConfig: NextConfig = {
  // Deploy and e2e run .next/standalone/server.js. next start warns once
  // this is set, and it is not the server the image runs.
  output: "standalone",
  cacheComponents: true,
  partialPrefetching: true,
  // Inlined at build time for the footer's copyright year: a prerendered
  // component can't call new Date(), and every deploy is a fresh build.
  env: {
    BUILD_YEAR: String(new Date().getFullYear()),
  },
  experimental: {
    agentUpgrade: "security",
  },
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
