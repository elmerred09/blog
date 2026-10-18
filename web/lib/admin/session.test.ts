// @vitest-environment node
import { beforeEach, describe, expect, it, vi } from "vitest";

import { getAuth } from "@/lib/auth";

import { requireAdmin } from "./session";

vi.mock("server-only", () => ({}));
vi.mock("next/headers", () => ({ headers: async () => new Headers() }));
vi.mock("next/navigation", () => ({
  notFound: () => {
    throw new Error("NOT_FOUND");
  },
  redirect: (url: string) => {
    throw new Error(`REDIRECT ${url}`);
  },
}));
vi.mock("@/lib/auth", () => ({ getAuth: vi.fn() }));

function signedIn(user: { githubId?: string | null; name: string } | null) {
  vi.mocked(getAuth).mockReturnValue({
    auth: {
      api: {
        getSession: vi.fn(async () =>
          user ? { user, session: { id: "s" } } : null,
        ),
      },
    },
    settings: { githubIds: new Set(["123"]) },
  } as unknown as ReturnType<typeof getAuth>);
}

describe("requireAdmin", () => {
  beforeEach(() => {
    vi.mocked(getAuth).mockReset();
  });

  it("returns the admin for an allowed GitHub id", async () => {
    signedIn({ githubId: "123", name: "Elmer" });
    await expect(requireAdmin()).resolves.toEqual({
      githubId: "123",
      name: "Elmer",
    });
  });

  it("sends a visitor with no session to sign in", async () => {
    signedIn(null);
    await expect(requireAdmin()).rejects.toThrow("REDIRECT /sign-in");
  });

  it("rejects a session without a GitHub id", async () => {
    signedIn({ githubId: null, name: "Elmer" });
    await expect(requireAdmin()).rejects.toThrow("REDIRECT /sign-in");
  });

  it("rejects a session whose id was removed from ADMIN_GITHUB_IDS", async () => {
    signedIn({ githubId: "456", name: "Former admin" });
    await expect(requireAdmin()).rejects.toThrow(
      "REDIRECT /sign-in?error=not_allowed",
    );
  });

  it("is a 404 when admin sign-in isn't configured", async () => {
    vi.mocked(getAuth).mockReturnValue(null);
    await expect(requireAdmin()).rejects.toThrow("NOT_FOUND");
  });
});
