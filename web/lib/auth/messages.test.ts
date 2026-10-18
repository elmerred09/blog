import { describe, expect, it } from "vitest";

import { signInErrorMessage } from "./messages";

describe("signInErrorMessage", () => {
  it.each([
    ["not_allowed", /isn't allowed/],
    ["not_configured", /isn't configured/],
    ["access_denied", /cancelled/],
    ["state_mismatch", /Sign-in failed/],
    ["<script>", /Sign-in failed/],
  ])("%s", (code, want) => {
    expect(signInErrorMessage(code)).toMatch(want);
  });
});
