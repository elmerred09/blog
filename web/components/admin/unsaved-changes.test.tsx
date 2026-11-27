import { fireEvent, render, screen } from "@testing-library/react";
import type { ComponentProps } from "react";
import { Activity, useState } from "react";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { savePost } from "@/app/admin/actions";
import { BackLink } from "@/components/admin/back-link";
import { PostEditor } from "@/components/admin/post-editor";
import { UnsavedChanges } from "@/components/admin/unsaved-changes";
import type { AdminPost } from "@/lib/api/types";

// Link only calls onNavigate when an app router is mounted, which these tests
// don't have. This stands in for that call.
vi.mock("next/link", () => ({
  default: ({
    href,
    children,
    onNavigate,
    ...props
  }: ComponentProps<"a"> & {
    href: string;
    onNavigate?: (event: { preventDefault: () => void }) => void;
  }) => (
    <a
      href={href}
      {...props}
      onClick={(event) => {
        // jsdom can't follow the href; the guard's preventDefault is what
        // these tests observe.
        event.preventDefault();
        onNavigate?.({
          preventDefault: () => event.preventDefault(),
        });
      }}
    >
      {children}
    </a>
  ),
}));
vi.mock("next/navigation", () => ({ useRouter: () => ({ replace: vi.fn() }) }));
vi.mock("@/app/admin/actions", () => ({
  savePost: vi.fn(),
  previewPost: vi.fn(),
  createImageUpload: vi.fn(),
}));
vi.mock("@/components/admin/editor", () => ({
  Editor: ({
    markdown,
    onChange,
  }: {
    markdown: { current: string };
    onChange: (md: string) => void;
  }) => {
    const [initial] = useState(() => markdown.current);
    return (
      <textarea
        aria-label="Body"
        defaultValue={initial}
        onChange={(event) => onChange(event.target.value)}
      />
    );
  },
}));

const post: AdminPost = {
  id: "0f8a5c1e-4b2d-4c3a-9e1f-2a3b4c5d6e7f",
  slug: "keyset-pagination",
  title: "Keyset pagination",
  summary: "Why OFFSET gets slow.",
  body_md: "Hello.",
  published_at: "2026-10-01T12:00:00Z",
  created_at: "2026-10-01T11:00:00Z",
  updated_at: "2026-10-18T12:00:12.345678Z",
  state: "published",
  tags: [{ slug: "postgres", name: "Postgres", color: "slate" }],
};

function renderEditor(current: AdminPost | null = null) {
  return render(
    <UnsavedChanges>
      <BackLink />
      <PostEditor post={current} tagSuggestions={["Postgres", "Go"]} />
    </UnsavedChanges>,
  );
}

function field(name: string): HTMLInputElement {
  return screen.getByRole<HTMLInputElement>("textbox", { name });
}

function back() {
  fireEvent.click(screen.getByRole("link", { name: "Back" }));
}

function dialog() {
  return screen.getByRole("dialog", { name: "Discard unsaved changes?" });
}

function expectNoDialog() {
  expect(
    screen.queryByRole("dialog", { name: "Discard unsaved changes?" }),
  ).toBeNull();
}

beforeEach(() => {
  vi.mocked(savePost).mockResolvedValue({ ok: true, value: post });
});

describe("Back link with unsaved changes", () => {
  it("leaves a clean editor without asking", () => {
    renderEditor();

    back();

    expectNoDialog();
  });

  it("asks when the title changed, and staying keeps the edit", () => {
    renderEditor();
    fireEvent.change(field("Title"), { target: { value: "Half written" } });

    back();

    expect(dialog()).toBeDefined();
    expect(document.activeElement).toBe(
      screen.getByRole("button", { name: "Keep editing" }),
    );
    fireEvent.click(screen.getByRole("button", { name: "Keep editing" }));

    expectNoDialog();
    expect(field("Title").value).toBe("Half written");
  });

  it("closes when Escape is pressed", () => {
    renderEditor(post);
    fireEvent.change(field("Summary"), { target: { value: "Edited." } });
    back();

    fireEvent(
      screen.getByRole("dialog"),
      new Event("cancel", { bubbles: true, cancelable: true }),
    );

    expectNoDialog();
    expect(field("Summary").value).toBe("Edited.");
  });

  it("closes when the backdrop is clicked", () => {
    renderEditor(post);
    fireEvent.change(field("Body"), { target: { value: "Changed." } });
    back();

    // jsdom gives the dialog no box, so a click away from the origin lands
    // outside it, which is the backdrop.
    fireEvent.click(screen.getByRole("dialog"), { clientX: 10, clientY: 10 });

    expectNoDialog();
  });

  it("asks for a new tag and for switching back to a draft", () => {
    renderEditor(post);

    fireEvent.change(screen.getByRole("combobox", { name: "Tags" }), {
      target: { value: "Go" },
    });
    fireEvent.keyDown(screen.getByRole("combobox", { name: "Tags" }), {
      key: "Enter",
    });
    back();
    expect(dialog()).toBeDefined();
    fireEvent.click(screen.getByRole("button", { name: "Keep editing" }));

    fireEvent.click(screen.getByRole("radio", { name: "Draft" }));
    back();
    expect(dialog()).toBeDefined();
  });

  // The route stays mounted while hidden, and a modal dialog would otherwise
  // still be open when it is shown again.
  it("closes the dialog when the page is hidden and shown again", () => {
    function InActivity({ hidden }: { hidden: boolean }) {
      return (
        <Activity mode={hidden ? "hidden" : "visible"}>
          <UnsavedChanges>
            <BackLink />
            <PostEditor post={null} tagSuggestions={[]} />
          </UnsavedChanges>
        </Activity>
      );
    }
    const { rerender } = render(<InActivity hidden={false} />);
    fireEvent.change(field("Title"), { target: { value: "Half written" } });
    back();
    expect(dialog()).toBeDefined();

    rerender(<InActivity hidden />);
    rerender(<InActivity hidden={false} />);

    expectNoDialog();
    expect(field("Title").value).toBe("Half written");
  });

  it("stops asking once the edit is reverted", () => {
    renderEditor();
    fireEvent.change(field("Title"), { target: { value: "Temporary" } });
    fireEvent.change(field("Title"), { target: { value: "" } });

    back();

    expectNoDialog();
  });

  it("stops asking after a successful save", async () => {
    renderEditor(post);
    fireEvent.change(field("Title"), { target: { value: "Saved title" } });
    fireEvent.click(screen.getByRole("button", { name: "Save" }));
    await screen.findByRole("status");

    back();

    expectNoDialog();
  });

  it("keeps asking when the save is rejected", async () => {
    vi.mocked(savePost).mockResolvedValueOnce({
      ok: false,
      error: { code: "conflict", message: "stale", problems: [] },
    });
    renderEditor(post);
    fireEvent.change(field("Title"), { target: { value: "Unsaved title" } });
    fireEvent.click(screen.getByRole("button", { name: "Save" }));
    await screen.findByRole("alert");

    back();

    expect(dialog()).toBeDefined();
    expect(
      screen
        .getByRole("link", { name: "Discard changes" })
        .getAttribute("href"),
    ).toBe("/admin");
  });

  it("ignores a schedule time left over after switching back", () => {
    renderEditor(post);
    fireEvent.click(screen.getByRole("radio", { name: "Schedule" }));
    fireEvent.change(screen.getByLabelText(/Publish at/), {
      target: { value: "2026-12-01T09:00" },
    });
    fireEvent.click(screen.getByRole("radio", { name: /Published / }));

    back();

    expectNoDialog();
  });
});
