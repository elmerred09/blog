import { render } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { PostBody } from "@/components/post-body";

describe("PostBody", () => {
  it("renders the rendered HTML inside the prose styles", () => {
    const { container } = render(
      <PostBody html='<h2>Plans</h2><pre class="chroma"><code>x</code></pre>' />,
    );

    const body = container.firstElementChild!;
    expect(body.classList).toContain("prose");
    expect(body.querySelector("h2")?.textContent).toBe("Plans");
    expect(body.querySelector("pre.chroma")).not.toBeNull();
  });
});
