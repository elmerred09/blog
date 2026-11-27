import {
  fireEvent,
  render,
  screen,
  waitFor,
  within,
} from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";

import { deletePost } from "@/app/admin/actions";
import { DeletePostButton } from "@/components/admin/delete-post-button";

const router = vi.hoisted(() => ({ replace: vi.fn(), refresh: vi.fn() }));
vi.mock("next/navigation", () => ({ useRouter: () => router }));
vi.mock("@/app/admin/actions", () => ({ deletePost: vi.fn() }));

const post = {
  id: "p1",
  slug: "keyset-pagination",
  title: "Keyset pagination",
};

function dialog() {
  return screen.queryByRole("dialog", { name: "Delete this post?" });
}

// The dialog's own Delete post button, not the one that opened it.
function confirm() {
  fireEvent.click(
    within(dialog()!).getByRole("button", { name: "Delete post" }),
  );
}

describe("DeletePostButton", () => {
  it("asks first, naming the post, and cancelling deletes nothing", () => {
    render(<DeletePostButton post={post} variant="text" after="list" />);

    fireEvent.click(screen.getByRole("button", { name: "Delete post" }));

    expect(dialog()?.textContent).toContain(
      "“Keyset pagination” leaves the site and the post list, along with any unsaved edits.",
    );
    fireEvent.click(screen.getByRole("button", { name: "Cancel" }));
    expect(dialog()).toBeNull();
    expect(deletePost).not.toHaveBeenCalled();
  });

  it("deletes on confirm and goes back to the post list", async () => {
    vi.mocked(deletePost).mockResolvedValue({ ok: true, value: undefined });
    render(<DeletePostButton post={post} variant="text" after="list" />);

    fireEvent.click(screen.getByRole("button", { name: "Delete post" }));
    confirm();

    await waitFor(() => expect(router.replace).toHaveBeenCalledWith("/admin"));
    expect(deletePost).toHaveBeenCalledWith({
      id: "p1",
      slug: "keyset-pagination",
    });
    expect(router.refresh).toHaveBeenCalled();
  });

  it("refreshes the page it is on when asked to", async () => {
    vi.mocked(deletePost).mockResolvedValue({ ok: true, value: undefined });
    render(<DeletePostButton post={post} variant="icon" after="refresh" />);

    fireEvent.click(
      screen.getByRole("button", { name: "Delete Keyset pagination" }),
    );
    confirm();

    await waitFor(() => expect(router.refresh).toHaveBeenCalled());
    expect(router.replace).not.toHaveBeenCalled();
    await waitFor(() => expect(dialog()).toBeNull());
  });

  it("keeps the dialog open with the reason when the delete fails", async () => {
    vi.mocked(deletePost).mockResolvedValue({
      ok: false,
      error: { code: "not_found", message: "post not found", problems: [] },
    });
    render(<DeletePostButton post={post} variant="text" after="list" />);

    fireEvent.click(screen.getByRole("button", { name: "Delete post" }));
    confirm();

    await waitFor(() =>
      expect(screen.getByRole("alert").textContent).toBe(
        "This post was already deleted. Reload to see the current list.",
      ),
    );
    expect(dialog()).not.toBeNull();
    expect(router.replace).not.toHaveBeenCalled();
  });

  it("says so when the action throws", async () => {
    vi.mocked(deletePost).mockRejectedValue(new Error("network"));
    render(<DeletePostButton post={post} variant="text" after="list" />);

    fireEvent.click(screen.getByRole("button", { name: "Delete post" }));
    confirm();

    await waitFor(() =>
      expect(screen.getByRole("alert").textContent).toBe(
        "Couldn't delete the post. Try again.",
      ),
    );
  });
});
