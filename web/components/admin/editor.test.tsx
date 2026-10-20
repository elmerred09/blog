import {
  act,
  fireEvent,
  render,
  screen,
  waitFor,
} from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";

import { Editor } from "@/components/admin/editor";
import type { UploadResult } from "@/lib/admin/upload";

function png(name = "query-plan.png"): File {
  return new File([new Uint8Array([1, 2, 3])], name, { type: "image/png" });
}

async function setup(
  initialMarkdown = "text",
  upload: (file: File) => Promise<UploadResult> = async () => ({
    ok: true,
    url: "http://s3.test/blog/images/abc.png",
  }),
) {
  const onChange = vi.fn();
  render(
    <Editor
      initialMarkdown={initialMarkdown}
      onChange={onChange}
      upload={upload}
    />,
  );
  // The editor mounts after the first render (immediatelyRender: false).
  const body = await screen.findByRole("textbox", { name: "Body" });
  return { onChange, upload, body };
}

function lastMarkdown(onChange: ReturnType<typeof vi.fn>): string {
  return String(onChange.mock.calls.at(-1)?.[0]).trim();
}

describe("Editor", () => {
  it("shows the loaded markdown", async () => {
    const { body } = await setup("## Plans\n\nSome **bold** text.");

    expect(body.querySelector("h2")?.textContent).toBe("Plans");
    expect(body.querySelector("strong")?.textContent).toBe("bold");
  });

  it("has a button for each toolbar feature", async () => {
    await setup();

    for (const name of [
      "Bold",
      "Italic",
      "Strikethrough",
      "Inline code",
      "Heading 2",
      "Heading 3",
      "Bullet list",
      "Numbered list",
      "Quote",
      "Link",
      "Code block",
      "Table",
      "Image",
      "Undo",
      "Redo",
    ]) {
      expect(screen.getByRole("button", { name })).toBeDefined();
    }
  });

  it("reports markdown after a toolbar change", async () => {
    const { onChange } = await setup();

    fireEvent.click(screen.getByRole("button", { name: "Heading 2" }));

    expect(lastMarkdown(onChange)).toBe("## text");
    expect(
      screen
        .getByRole("button", { name: "Heading 2" })
        .getAttribute("aria-pressed"),
    ).toBe("true");
  });

  it("sets a code block's language", async () => {
    const { onChange } = await setup();

    fireEvent.click(screen.getByRole("button", { name: "Code block" }));
    fireEvent.change(screen.getByRole("textbox", { name: "Code language" }), {
      target: { value: "go" },
    });

    expect(lastMarkdown(onChange)).toBe("```go\ntext\n```");
  });

  it("shows table controls inside a table", async () => {
    await setup("");

    expect(screen.queryByRole("button", { name: "Add row" })).toBeNull();
    fireEvent.click(screen.getByRole("button", { name: "Table" }));

    expect(screen.getByRole("button", { name: "Add row" })).toBeDefined();
    expect(screen.getByRole("button", { name: "Delete table" })).toBeDefined();
  });

  it("uploads a chosen image and inserts it", async () => {
    const upload = vi.fn(async () => ({
      ok: true as const,
      url: "http://s3.test/blog/images/abc.png",
    }));
    const { onChange } = await setup("", upload);

    const file = png();
    await act(async () => {
      fireEvent.change(screen.getByLabelText("Upload image"), {
        target: { files: [file] },
      });
    });

    expect(upload).toHaveBeenCalledExactlyOnceWith(file);
    await waitFor(() =>
      expect(lastMarkdown(onChange)).toBe(
        "![query plan](http://s3.test/blog/images/abc.png)",
      ),
    );
  });

  it("uploads a pasted image", async () => {
    const upload = vi.fn(async () => ({
      ok: true as const,
      url: "http://s3.test/blog/images/abc.png",
    }));
    const { body } = await setup("", upload);

    const file = png();
    await act(async () => {
      fireEvent.paste(body, {
        clipboardData: {
          files: [file],
          items: [],
          types: ["Files"],
          getData: () => "",
        },
      });
    });

    expect(upload).toHaveBeenCalledExactlyOnceWith(file);
  });

  it("shows a failed upload without inserting anything", async () => {
    const upload = vi.fn(async () => ({
      ok: false as const,
      message: "The image store refused the file.",
    }));
    const { onChange } = await setup("", upload);

    await act(async () => {
      fireEvent.change(screen.getByLabelText("Upload image"), {
        target: { files: [png()] },
      });
    });

    expect((await screen.findByRole("alert")).textContent).toContain(
      "The image store refused the file.",
    );
    expect(onChange).not.toHaveBeenCalled();
  });
});
