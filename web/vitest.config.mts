import react from "@vitejs/plugin-react";
import { configDefaults, defineConfig } from "vitest/config";

export default defineConfig({
  plugins: [react()],
  resolve: { tsconfigPaths: true },
  test: {
    environment: "jsdom",
    setupFiles: ["./vitest.setup.ts"],
    // Playwright's specs; they'd match Vitest's default include.
    exclude: [...configDefaults.exclude, "e2e/**"],
    clearMocks: true,
    unstubEnvs: true,
    unstubGlobals: true,
  },
});
