import { act, fireEvent, render, screen } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import { PostBody } from "@/components/post-body";

// goldmark + chroma's shape: one span per line, each ending in a newline.
const goBlock =
  '<pre class="chroma"><code><span class="line"><span class="cl">' +
  '<span class="kd">func</span> main() {}\n</span></span>' +
  '<span class="line"><span class="cl">// done\n</span></span></code></pre>';

describe("PostBody", () => {
  let writeText: ReturnType<typeof vi.fn>;

  beforeEach(() => {
    writeText = vi.fn().mockResolvedValue(undefined);
    Object.defineProperty(navigator, "clipboard", {
      value: { writeText },
      configurable: true,
    });
  });

  afterEach(() => {
    vi.useRealTimers();
  });

  it("renders the rendered HTML inside the prose styles", () => {
    const { container } = render(
      <PostBody html={`<h2>Plans</h2>${goBlock}`} />,
    );

    const body = container.firstElementChild!;
    expect(body.classList).toContain("prose");
    expect(body.querySelector("h2")?.textContent).toBe("Plans");
    expect(body.querySelector("pre.chroma")).not.toBeNull();
  });

  it("adds a copy button to each code block, not to inline code", () => {
    render(
      <PostBody html={`<p>Use <code>ctx</code>.</p>${goBlock}${goBlock}`} />,
    );

    expect(screen.getAllByRole("button", { name: "Copy code" })).toHaveLength(
      2,
    );
  });

  it("copies the block's text without the trailing newline", async () => {
    render(<PostBody html={goBlock} />);

    await act(async () => {
      fireEvent.click(screen.getByRole("button", { name: "Copy code" }));
    });

    expect(writeText).toHaveBeenCalledExactlyOnceWith(
      "func main() {}\n// done",
    );
    expect(screen.getByRole("button").textContent).toBe("Copied");
  });

  it("goes back to Copy after a moment", async () => {
    vi.useFakeTimers();
    render(<PostBody html={goBlock} />);

    await act(async () => {
      fireEvent.click(screen.getByRole("button", { name: "Copy code" }));
    });
    expect(screen.getByRole("button").textContent).toBe("Copied");

    act(() => vi.advanceTimersByTime(2000));
    expect(screen.getByRole("button").textContent).toBe("Copy");
  });

  it("says so when the clipboard refuses", async () => {
    writeText.mockRejectedValue(new DOMException("denied", "NotAllowedError"));
    render(<PostBody html={goBlock} />);

    await act(async () => {
      fireEvent.click(screen.getByRole("button", { name: "Copy code" }));
    });

    expect(screen.getByRole("button").textContent).toBe("Copy failed");
  });

  it("follows new HTML, as the editor's preview does", () => {
    const { rerender } = render(<PostBody html={goBlock} />);
    expect(screen.getAllByRole("button")).toHaveLength(1);

    rerender(<PostBody html={`${goBlock}<p>and</p>${goBlock}`} />);
    expect(screen.getAllByRole("button")).toHaveLength(2);

    rerender(<PostBody html="<p>No code.</p>" />);
    expect(screen.queryByRole("button")).toBeNull();
  });
});
