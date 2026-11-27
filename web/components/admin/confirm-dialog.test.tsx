import { fireEvent, render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";

import { ConfirmDialog } from "@/components/admin/confirm-dialog";

function renderDialog(
  props: Partial<Parameters<typeof ConfirmDialog>[0]> = {},
) {
  const onCancel = vi.fn();
  const onConfirm = vi.fn();
  render(
    <ConfirmDialog
      title="Delete this post?"
      description="It leaves the site."
      confirmLabel="Delete post"
      confirm={{ onConfirm }}
      cancelLabel="Cancel"
      onCancel={onCancel}
      pendingLabel="Deleting…"
      {...props}
    />,
  );
  return { onCancel, onConfirm };
}

function dialog() {
  return screen.getByRole("dialog", { name: "Delete this post?" });
}

// jsdom gives the dialog no box, so a click away from the origin lands outside
// it, which is the backdrop.
describe("ConfirmDialog", () => {
  it("opens with the safe choice focused and the text described", () => {
    renderDialog();

    expect(dialog().getAttribute("aria-describedby")).toBeTruthy();
    expect(screen.getByText("It leaves the site.")).toBeDefined();
    expect(document.activeElement).toBe(
      screen.getByRole("button", { name: "Cancel" }),
    );
  });

  it("runs onConfirm from the confirm button", () => {
    const { onConfirm, onCancel } = renderDialog();

    fireEvent.click(screen.getByRole("button", { name: "Delete post" }));

    expect(onConfirm).toHaveBeenCalledOnce();
    expect(onCancel).not.toHaveBeenCalled();
  });

  it.each([
    [
      "the cancel button",
      () => fireEvent.click(screen.getByRole("button", { name: "Cancel" })),
    ],
    [
      "Escape",
      () => fireEvent(dialog(), new Event("cancel", { cancelable: true })),
    ],
    [
      "a click on the backdrop",
      () => fireEvent.click(dialog(), { clientX: 10, clientY: 10 }),
    ],
  ])("cancels on %s", (_, act) => {
    const { onCancel } = renderDialog();

    act();

    expect(onCancel).toHaveBeenCalledOnce();
  });

  it("can't be dismissed while the action runs", () => {
    const { onCancel } = renderDialog({ pending: true });

    const confirm = screen.getByRole<HTMLButtonElement>("button", {
      name: "Deleting…",
    });
    expect(confirm.disabled).toBe(true);
    fireEvent(dialog(), new Event("cancel", { cancelable: true }));
    fireEvent.click(dialog(), { clientX: 10, clientY: 10 });

    expect(onCancel).not.toHaveBeenCalled();
  });

  it("shows an error", () => {
    renderDialog({ error: "post not found" });

    expect(screen.getByRole("alert").textContent).toBe("post not found");
  });

  it("can confirm with a link", () => {
    renderDialog({
      confirm: { href: "/admin" },
      confirmLabel: "Discard changes",
    });

    expect(
      screen
        .getByRole("link", { name: "Discard changes" })
        .getAttribute("href"),
    ).toBe("/admin");
  });
});
