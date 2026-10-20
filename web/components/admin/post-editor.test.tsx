import {
  act,
  fireEvent,
  render,
  screen,
  waitFor,
} from "@testing-library/react";
import { Activity } from "react";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { createImageUpload, previewPost, savePost } from "@/app/admin/actions";
import { PostEditor } from "@/components/admin/post-editor";
import type { AdminPost } from "@/lib/api/types";

const replace = vi.fn();
vi.mock("next/navigation", () => ({ useRouter: () => ({ replace }) }));
vi.mock("@/app/admin/actions", () => ({
  savePost: vi.fn(),
  previewPost: vi.fn(),
  createImageUpload: vi.fn(),
}));
// The real editor is covered by editor.test.tsx; a textarea stands in here.
vi.mock("@/components/admin/editor", () => ({
  Editor: ({
    initialMarkdown,
    onChange,
  }: {
    initialMarkdown: string;
    onChange: (md: string) => void;
  }) => (
    <textarea
      aria-label="Body"
      defaultValue={initialMarkdown}
      onChange={(e) => onChange(e.target.value)}
    />
  ),
}));

const LOCK = "2026-10-18T12:00:12.345678Z";

const post: AdminPost = {
  id: "0f8a5c1e-4b2d-4c3a-9e1f-2a3b4c5d6e7f",
  slug: "keyset-pagination",
  title: "Keyset pagination",
  summary: "Why OFFSET gets slow.",
  body_md: "Hello.",
  published_at: "2026-10-01T12:00:00Z",
  created_at: "2026-10-01T11:00:00Z",
  updated_at: LOCK,
  state: "published",
  tags: [{ slug: "postgres", name: "Postgres" }],
};

function saved(overrides: Partial<AdminPost>) {
  return { ok: true as const, value: { ...post, ...overrides } };
}

function field(name: string): HTMLInputElement {
  return screen.getByRole<HTMLInputElement>("textbox", { name });
}

function type(name: string, value: string) {
  fireEvent.change(field(name), { target: { value } });
}

async function save() {
  await act(async () => {
    fireEvent.click(screen.getByRole("button", { name: "Save" }));
  });
}

beforeEach(() => {
  vi.mocked(savePost).mockResolvedValue(saved({}));
});

describe("PostEditor, new post", () => {
  it("suggests a slug from the title", () => {
    render(<PostEditor post={null} tagSuggestions={[]} />);

    type("Title", "Keyset Pagination in Postgres");

    expect(field("Slug").value).toBe("keyset-pagination-in-postgres");
  });

  it("stops following the title once the slug is edited", () => {
    render(<PostEditor post={null} tagSuggestions={[]} />);

    type("Title", "First");
    type("Slug", "my-slug");
    type("Title", "Second");

    expect(field("Slug").value).toBe("my-slug");
  });

  it("creates the post as a draft and opens its editor", async () => {
    render(<PostEditor post={null} tagSuggestions={[]} />);
    type("Title", "Keyset pagination");
    type("Summary", "Why OFFSET gets slow.");
    type("Body", "Hello.");

    await save();

    expect(savePost).toHaveBeenCalledExactlyOnceWith({
      id: null,
      loadedSlug: null,
      updatedAt: null,
      post: {
        slug: "keyset-pagination",
        title: "Keyset pagination",
        summary: "Why OFFSET gets slow.",
        body_md: "Hello.",
        published_at: null,
        tags: [],
      },
    });
    expect(replace).toHaveBeenCalledExactlyOnceWith(`/admin/posts/${post.id}`);
  });

  // Next keeps visited routes mounted but hidden in <Activity>, so after a
  // create navigates away, /admin/posts/new would come back still bound to
  // the post it created.
  function InActivity({ hidden }: { hidden: boolean }) {
    return (
      <Activity mode={hidden ? "hidden" : "visible"}>
        <PostEditor post={null} tagSuggestions={[]} />
      </Activity>
    );
  }

  it("starts blank when shown again after creating a post", async () => {
    const { rerender } = render(<InActivity hidden={false} />);
    type("Title", "Keyset pagination");
    await save();

    rerender(<InActivity hidden />);
    rerender(<InActivity hidden={false} />);

    expect(field("Title").value).toBe("");
    expect(screen.queryByRole("status")).toBeNull();
    vi.mocked(savePost).mockClear();
    type("Title", "Another post");
    await save();
    expect(vi.mocked(savePost).mock.calls[0][0]).toMatchObject({
      id: null,
      updatedAt: null,
    });
  });

  it("keeps an unsaved draft when hidden and shown again", () => {
    const { rerender } = render(<InActivity hidden={false} />);
    type("Title", "Half written");

    rerender(<InActivity hidden />);
    rerender(<InActivity hidden={false} />);

    expect(field("Title").value).toBe("Half written");
  });
});

describe("PostEditor, existing post", () => {
  it("doesn't change the slug when the title changes", () => {
    render(<PostEditor post={post} tagSuggestions={[]} />);

    type("Title", "Something else");

    expect(field("Slug").value).toBe(post.slug);
  });

  it("sends the loaded lock token, then the one each save returns", async () => {
    const next = "2026-10-18T12:05:00.000001Z";
    vi.mocked(savePost).mockResolvedValueOnce(saved({ updated_at: next }));
    render(<PostEditor post={post} tagSuggestions={[]} />);

    await save();
    await save();

    const calls = vi.mocked(savePost).mock.calls.map(([input]) => input);
    expect(calls.map((c) => c.updatedAt)).toEqual([LOCK, next]);
    expect(calls[0]).toMatchObject({
      id: post.id,
      loadedSlug: post.slug,
      post: { published_at: post.published_at, tags: ["Postgres"] },
    });
    expect(replace).not.toHaveBeenCalled();
  });

  it("sends the slug it loaded, then the one it saved", async () => {
    vi.mocked(savePost).mockResolvedValueOnce(saved({ slug: "renamed" }));
    render(<PostEditor post={post} tagSuggestions={[]} />);
    type("Slug", "renamed");

    await save();
    await save();

    const loaded = vi
      .mocked(savePost)
      .mock.calls.map(([input]) => input.loadedSlug);
    expect(loaded).toEqual([post.slug, "renamed"]);
  });

  it("warns before renaming a published post", () => {
    render(<PostEditor post={post} tagSuggestions={[]} />);

    expect(screen.queryByText(/breaks links/)).toBeNull();
    type("Slug", "renamed");

    expect(screen.getByText(/breaks links/)).toBeDefined();
  });

  it("explains a conflict", async () => {
    vi.mocked(savePost).mockResolvedValueOnce({
      ok: false,
      error: { code: "conflict", message: "stale", problems: [] },
    });
    render(<PostEditor post={post} tagSuggestions={[]} />);

    await save();

    expect(screen.getByRole("alert").textContent).toMatch(/changed since/);
  });

  it("puts a taken slug on the slug field", async () => {
    vi.mocked(savePost).mockResolvedValueOnce({
      ok: false,
      error: { code: "slug_taken", message: "taken", problems: [] },
    });
    render(<PostEditor post={post} tagSuggestions={[]} />);

    await save();

    expect(field("Slug").getAttribute("aria-invalid")).toBe("true");
    expect(screen.getByText(/already uses this slug/)).toBeDefined();
  });

  it("shows each problem next to its field", async () => {
    vi.mocked(savePost).mockResolvedValueOnce({
      ok: false,
      error: {
        code: "invalid_request",
        message: "title is required; summary is required",
        problems: [
          { field: "title", message: "title is required" },
          { field: "summary", message: "summary is required" },
        ],
      },
    });
    render(<PostEditor post={post} tagSuggestions={[]} />);

    await save();

    expect(field("Title").getAttribute("aria-invalid")).toBe("true");
    expect(screen.getByText("title is required")).toBeDefined();
    expect(screen.getByText("summary is required")).toBeDefined();
  });

  it("refuses a schedule in the past without saving", async () => {
    render(<PostEditor post={post} tagSuggestions={[]} />);
    fireEvent.click(screen.getByRole("radio", { name: "Schedule" }));
    fireEvent.change(screen.getByLabelText(/Publish at/), {
      target: { value: "2020-01-01T09:00" },
    });

    await save();

    expect(savePost).not.toHaveBeenCalled();
    expect(screen.getByText(/in the future/)).toBeDefined();
  });

  it("unpublishes by saving as a draft", async () => {
    render(<PostEditor post={post} tagSuggestions={[]} />);
    fireEvent.click(screen.getByRole("radio", { name: "Draft" }));

    await save();

    expect(vi.mocked(savePost).mock.calls[0][0].post.published_at).toBeNull();
  });

  it("previews the current body", async () => {
    vi.mocked(previewPost).mockResolvedValue({
      ok: true,
      value: "<p>Changed.</p>",
    });
    render(<PostEditor post={post} tagSuggestions={[]} />);
    type("Body", "Changed.");

    await act(async () => {
      fireEvent.click(screen.getByRole("tab", { name: "Preview" }));
    });

    expect(previewPost).toHaveBeenCalledExactlyOnceWith("Changed.");
    await waitFor(() => expect(screen.getByText("Changed.")).toBeDefined());
    expect(createImageUpload).not.toHaveBeenCalled();
  });
});
