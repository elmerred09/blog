import {
  act,
  fireEvent,
  render,
  screen,
  waitFor,
} from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";

import { setTagColor } from "@/app/admin/actions";
import { TagColorList } from "@/components/admin/tag-color-list";
import type { Tag } from "@/lib/api";

vi.mock("@/app/admin/actions", () => ({ setTagColor: vi.fn() }));

const tags: Tag[] = [
  { slug: "go", name: "Go", color: "slate" },
  { slug: "postgres", name: "Postgres", color: "teal" },
];

function swatch(tag: string, color: string): HTMLInputElement {
  return screen
    .getByRole("group", { name: `Color for ${tag}` })
    .querySelector<HTMLInputElement>(`input[value="${color}"]`)!;
}

type SaveResult = Awaited<ReturnType<typeof setTagColor>>;

// Each call to setTagColor returns a promise the test settles, in any order.
function deferSaves() {
  const pending: ((r: SaveResult) => void)[] = [];
  vi.mocked(setTagColor).mockImplementation(
    () => new Promise<SaveResult>((resolve) => pending.push(resolve)),
  );
  return async (i: number, result: SaveResult) => {
    await act(async () => pending[i](result));
  };
}

const goIn = (color: Tag["color"]): SaveResult => ({
  ok: true,
  value: { slug: "go", name: "Go", color },
});
const failed: SaveResult = {
  ok: false,
  error: { code: "not_found", message: "tag not found", problems: [] },
};

function chip(name: string): HTMLElement {
  return screen.getByText(name, { selector: "span" });
}

describe("TagColorList", () => {
  it("shows each tag in its color with that color chosen", () => {
    render(<TagColorList tags={tags} />);

    expect(swatch("Go", "slate").checked).toBe(true);
    expect(swatch("Postgres", "teal").checked).toBe(true);
    expect(chip("Postgres").className).toContain("bg-tag-teal");
    const goSwatches = screen
      .getByRole("group", { name: "Color for Go" })
      .querySelectorAll("input[type=radio]");
    expect(Array.from(goSwatches, (r) => r.getAttribute("aria-label"))).toEqual(
      [
        "Slate",
        "Coral",
        "Rust",
        "Amber",
        "Yellow",
        "Olive",
        "Brown",
        "Forest",
        "Green",
        "Teal",
        "Petrol",
        "Blue",
        "Navy",
        "Indigo",
        "Purple",
        "Pink",
      ],
    );
  });

  it("saves a chosen color and shows it on the chip", async () => {
    vi.mocked(setTagColor).mockResolvedValue({
      ok: true,
      value: { slug: "go", name: "Go", color: "amber" },
    });
    render(<TagColorList tags={tags} />);

    fireEvent.click(swatch("Go", "amber"));

    expect(chip("Go").className).toContain("bg-tag-amber");
    await waitFor(() =>
      expect(setTagColor).toHaveBeenCalledWith({ slug: "go", color: "amber" }),
    );
    await waitFor(() => expect(screen.getByText("Saved")).toBeDefined());
  });

  it("puts the saved color back and says why when the save fails", async () => {
    vi.mocked(setTagColor).mockResolvedValue({
      ok: false,
      error: { code: "not_found", message: "tag not found", problems: [] },
    });
    render(<TagColorList tags={tags} />);

    fireEvent.click(swatch("Go", "coral"));

    await waitFor(() =>
      expect(screen.getByRole("alert").textContent).toBe("tag not found"),
    );
    expect(swatch("Go", "slate").checked).toBe(true);
    expect(chip("Go").className).toContain("bg-tag-slate");
  });

  it("puts the saved color back when the action throws", async () => {
    vi.mocked(setTagColor).mockRejectedValue(new Error("network"));
    render(<TagColorList tags={tags} />);

    fireEvent.click(swatch("Go", "coral"));

    await waitFor(() =>
      expect(screen.getByRole("alert").textContent).toBe(
        "Couldn't save the color. Try again.",
      ),
    );
    expect(swatch("Go", "slate").checked).toBe(true);
  });

  it("keeps the latest pick when an earlier save fails after it", async () => {
    const settle = deferSaves();
    render(<TagColorList tags={tags} />);

    fireEvent.click(swatch("Go", "amber"));
    fireEvent.click(swatch("Go", "teal"));
    await settle(0, failed);

    expect(chip("Go").className).toContain("bg-tag-teal");
    expect(swatch("Go", "teal").checked).toBe(true);
    expect(screen.queryByRole("alert")).toBeNull();

    await settle(1, goIn("teal"));

    expect(chip("Go").className).toContain("bg-tag-teal");
    expect(screen.getByText("Saved")).toBeDefined();
  });

  it("says Saving until the latest pick is saved", async () => {
    const settle = deferSaves();
    render(<TagColorList tags={tags} />);

    fireEvent.click(swatch("Go", "amber"));
    fireEvent.click(swatch("Go", "teal"));
    await settle(0, goIn("amber"));

    expect(screen.getByText("Saving…")).toBeDefined();
    expect(chip("Go").className).toContain("bg-tag-teal");
  });

  it("falls back to an earlier pick that saved when the latest fails", async () => {
    const settle = deferSaves();
    render(<TagColorList tags={tags} />);

    fireEvent.click(swatch("Go", "amber"));
    fireEvent.click(swatch("Go", "teal"));
    await settle(0, goIn("amber"));
    await settle(1, failed);

    expect(chip("Go").className).toContain("bg-tag-amber");
    expect(swatch("Go", "amber").checked).toBe(true);
    expect(screen.getByRole("alert").textContent).toBe("tag not found");
  });

  it("says when there are no tags yet", () => {
    render(<TagColorList tags={[]} />);

    expect(screen.getByText(/No tags yet/)).toBeDefined();
  });
});
