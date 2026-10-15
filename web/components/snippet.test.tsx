import { render } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { Snippet } from "@/components/snippet";

describe("Snippet", () => {
  it("wraps matches in mark and keeps the text in order", () => {
    const { container } = render(
      <Snippet
        fragments={[
          { text: "use ", match: false },
          { text: "keyset", match: true },
          { text: " pagination, not ", match: false },
          { text: "OFFSET", match: true },
        ]}
      />,
    );

    expect(container.textContent).toBe("use keyset pagination, not OFFSET");
    const marks = [...container.querySelectorAll("mark")].map(
      (m) => m.textContent,
    );
    expect(marks).toEqual(["keyset", "OFFSET"]);
  });

  it("renders HTML in the text as text", () => {
    const { container } = render(
      <Snippet
        fragments={[
          { text: "<script>alert(1)</script> ", match: false },
          { text: "<b>bold</b>", match: true },
        ]}
      />,
    );

    expect(container.querySelector("script")).toBeNull();
    expect(container.querySelector("b")).toBeNull();
    expect(container.textContent).toBe("<script>alert(1)</script> <b>bold</b>");
  });

  it("renders nothing for an empty snippet", () => {
    const { container } = render(<Snippet fragments={[]} />);

    expect(container.innerHTML).toBe("");
  });
});
