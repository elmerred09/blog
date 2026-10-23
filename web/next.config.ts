import type { NextConfig } from "next";

const nextConfig: NextConfig = {
  // Deploy and e2e run .next/standalone/server.js. next start warns once
  // this is set, and it is not the server the image runs.
  output: "standalone",
  cacheComponents: true,
  partialPrefetching: true,
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
