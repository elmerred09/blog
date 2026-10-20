import { fireEvent, render, screen } from "@testing-library/react";
import { useState } from "react";
import { describe, expect, it, vi } from "vitest";

import { TagInput } from "@/components/admin/tag-input";

function Harness({
  initial = [],
  suggestions = ["Postgres", "Go", "Search"],
  onChange = () => {},
}: {
  initial?: string[];
  suggestions?: string[];
  onChange?: (tags: string[]) => void;
}) {
  const [tags, setTags] = useState(initial);
  return (
    <TagInput
      value={tags}
      suggestions={suggestions}
      onChange={(next) => {
        setTags(next);
        onChange(next);
      }}
    />
  );
}

function input(): HTMLInputElement {
  return screen.getByRole<HTMLInputElement>("combobox", { name: "Tags" });
}

function type(text: string) {
  fireEvent.change(input(), { target: { value: text } });
}

function chips(): string[] {
  return screen
    .queryAllByRole("listitem")
    .map((li) => li.firstChild?.textContent ?? "");
}

describe("TagInput", () => {
  it("adds a tag on Enter", () => {
    render(<Harness />);

    type("  Postgres internals ");
    fireEvent.keyDown(input(), { key: "Enter" });

    expect(chips()).toEqual(["Postgres internals"]);
    expect(input().value).toBe("");
  });

  it("adds a tag on a comma", () => {
    render(<Harness />);

    type("Go");
    fireEvent.keyDown(input(), { key: "," });

    expect(chips()).toEqual(["Go"]);
  });

  it("adds tags pasted or typed with commas, keeping the rest as text", () => {
    render(<Harness initial={["Go"]} />);

    type("Postgres, go,  Search,Inde");

    expect(chips()).toEqual(["Go", "Postgres", "Search"]);
    expect(input().value).toBe("Inde");
  });

  it("adds a pending tag when focus leaves", () => {
    render(<Harness />);

    type("Search");
    fireEvent.blur(input());

    expect(chips()).toEqual(["Search"]);
  });

  it("keeps Enter from submitting the form", () => {
    render(<Harness />);

    type("Go");
    const event = new KeyboardEvent("keydown", {
      key: "Enter",
      bubbles: true,
      cancelable: true,
    });
    input().dispatchEvent(event);

    expect(event.defaultPrevented).toBe(true);
  });

  it("ignores a blank tag", () => {
    const onChange = vi.fn();
    render(<Harness onChange={onChange} />);

    type("   ");
    fireEvent.keyDown(input(), { key: "Enter" });

    expect(onChange).not.toHaveBeenCalled();
  });

  it("ignores a tag that would have the same slug as one already added", () => {
    const onChange = vi.fn();
    render(<Harness initial={["Postgres"]} onChange={onChange} />);

    type("postgres!");
    fireEvent.keyDown(input(), { key: "Enter" });

    expect(chips()).toEqual(["Postgres"]);
    expect(onChange).not.toHaveBeenCalled();
    expect(input().value).toBe("");
  });

  it("removes a tag with its button", () => {
    render(<Harness initial={["Go", "Postgres"]} />);

    fireEvent.click(screen.getByRole("button", { name: "Remove Go" }));

    expect(chips()).toEqual(["Postgres"]);
  });

  it("removes the last tag on Backspace in an empty input", () => {
    render(<Harness initial={["Go", "Postgres"]} />);

    fireEvent.keyDown(input(), { key: "Backspace" });

    expect(chips()).toEqual(["Go"]);
  });

  it("keeps the tags on Backspace while typing", () => {
    render(<Harness initial={["Go"]} />);

    type("Po");
    fireEvent.keyDown(input(), { key: "Backspace" });

    expect(chips()).toEqual(["Go"]);
  });

  it("suggests existing tags not already added", () => {
    const { container } = render(<Harness initial={["Go"]} />);

    const options = [...container.querySelectorAll("datalist option")].map(
      (o) => o.getAttribute("value"),
    );
    expect(options).toEqual(["Postgres", "Search"]);
  });
});
