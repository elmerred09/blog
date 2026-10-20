import { Editor } from "@tiptap/core";
import { afterEach, describe, expect, it } from "vitest";

import { editorExtensions } from "./editor-extensions";

// These check the extension setup: that each toolbar feature comes out as
// the markdown goldmark renders (GFM tables, fenced code with its language),
// not that every possible document round-trips unchanged.

let editor: Editor | undefined;

afterEach(() => {
  editor?.destroy();
  editor = undefined;
});

function load(markdown: string): Editor {
  editor = new Editor({
    extensions: editorExtensions(),
    content: markdown,
    contentType: "markdown",
  });
  return editor;
}

// Selects the whole paragraph "text" and applies a toolbar command.
function apply(command: (e: Editor) => void): string {
  const e = load("text");
  e.commands.selectAll();
  command(e);
  return e.getMarkdown().trim();
}

describe("toolbar markdown", () => {
  it.each<[string, (e: Editor) => void, string]>([
    ["bold", (e) => e.commands.toggleBold(), "**text**"],
    ["italic", (e) => e.commands.toggleItalic(), "*text*"],
    ["strikethrough", (e) => e.commands.toggleStrike(), "~~text~~"],
    ["inline code", (e) => e.commands.toggleCode(), "`text`"],
    ["heading 2", (e) => e.commands.toggleHeading({ level: 2 }), "## text"],
    ["heading 3", (e) => e.commands.toggleHeading({ level: 3 }), "### text"],
    ["bullet list", (e) => e.commands.toggleBulletList(), "- text"],
    ["ordered list", (e) => e.commands.toggleOrderedList(), "1. text"],
    ["quote", (e) => e.commands.toggleBlockquote(), "> text"],
    [
      "link",
      (e) => e.commands.setLink({ href: "https://example.com" }),
      "[text](https://example.com)",
    ],
    [
      "code block with a language",
      (e) => e.commands.toggleCodeBlock({ language: "go" }),
      "```go\ntext\n```",
    ],
  ])("writes %s", (_, command, want) => {
    expect(apply(command)).toBe(want);
  });

  it("writes an image", () => {
    const e = load("");
    e.commands.setImage({
      src: "http://s3.test/blog/images/abc.png",
      alt: "query plan",
    });

    expect(e.getMarkdown().trim()).toBe(
      "![query plan](http://s3.test/blog/images/abc.png)",
    );
  });

  it("writes a table as GFM", () => {
    const e = load("");
    e.commands.insertTable({ rows: 2, cols: 2, withHeaderRow: true });

    const lines = e.getMarkdown().trim().split("\n");
    expect(lines).toHaveLength(3);
    expect(lines[1]).toMatch(/^\|\s*-{3,}\s*\|\s*-{3,}\s*\|$/);
  });

  it("has no underline, which markdown can't express", () => {
    const e = load("text");

    expect(e.extensionManager.extensions.map((x) => x.name)).not.toContain(
      "underline",
    );
  });
});

describe("loading markdown", () => {
  const post = [
    "## Heading",
    "",
    "Some **bold**, *italic*, ~~struck~~ and `code`, with a [link](https://example.com).",
    "",
    "- one",
    "  - nested",
    "",
    "1. first",
    "",
    "> quoted",
    "",
    "```sql",
    "SELECT 1;",
    "```",
    "",
    "![alt](http://s3.test/blog/images/abc.png)",
  ].join("\n");

  it("writes back what it loaded", () => {
    expect(load(post).getMarkdown().trim()).toBe(post);
  });

  it("keeps a code block's language and contents verbatim", () => {
    const code = "```go\nif a < b && c > d {\n\treturn *p\n}\n```";

    expect(load(code).getMarkdown().trim()).toBe(code);
  });
});
