import { fireEvent, render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";

import { PublishControls } from "@/components/admin/publish-controls";
import type { PublishState } from "@/lib/admin/publish";

const draft: PublishState = { mode: "draft", scheduleLocal: "" };

describe("PublishControls", () => {
  it("offers draft, publish now, and schedule", () => {
    render(
      <PublishControls value={draft} onChange={() => {}} publishedAt={null} />,
    );

    expect(
      screen.getByRole<HTMLInputElement>("radio", { name: "Draft" }).checked,
    ).toBe(true);
    expect(screen.getByRole("radio", { name: "Publish now" })).toBeDefined();
    expect(screen.getByRole("radio", { name: "Schedule" })).toBeDefined();
  });

  it("hides the time picker unless scheduling", () => {
    render(
      <PublishControls value={draft} onChange={() => {}} publishedAt={null} />,
    );

    expect(screen.queryByLabelText(/Publish at/)).toBeNull();
  });

  it("reports a mode change", () => {
    const onChange = vi.fn();
    render(
      <PublishControls value={draft} onChange={onChange} publishedAt={null} />,
    );

    fireEvent.click(screen.getByRole("radio", { name: "Schedule" }));

    expect(onChange).toHaveBeenCalledExactlyOnceWith({
      mode: "schedule",
      scheduleLocal: "",
    });
  });

  it("reports the scheduled local time", () => {
    const onChange = vi.fn();
    render(
      <PublishControls
        value={{ mode: "schedule", scheduleLocal: "" }}
        onChange={onChange}
        publishedAt={null}
      />,
    );

    fireEvent.change(screen.getByLabelText(/Publish at/), {
      target: { value: "2026-10-20T09:30" },
    });

    expect(onChange).toHaveBeenCalledExactlyOnceWith({
      mode: "schedule",
      scheduleLocal: "2026-10-20T09:30",
    });
  });

  it("names the time zone the picker uses", () => {
    render(
      <PublishControls
        value={{ mode: "schedule", scheduleLocal: "" }}
        onChange={() => {}}
        publishedAt={null}
      />,
    );

    const zone = Intl.DateTimeFormat().resolvedOptions().timeZone;
    expect(
      screen.getByLabelText(/Publish at/).closest("label")?.textContent,
    ).toContain(zone);
  });

  it("says a published post keeps its date", () => {
    render(
      <PublishControls
        value={{ mode: "now", scheduleLocal: "" }}
        onChange={() => {}}
        publishedAt="2026-10-01T12:00:00Z"
        now={new Date("2026-10-18T16:00:00Z")}
      />,
    );

    expect(screen.getByRole("radio", { name: /Published/ })).toBeDefined();
    expect(screen.queryByRole("radio", { name: "Publish now" })).toBeNull();
  });

  it("shows an error", () => {
    render(
      <PublishControls
        value={draft}
        onChange={() => {}}
        publishedAt={null}
        error="Pick a time in the future."
      />,
    );

    expect(screen.getByRole("alert").textContent).toBe(
      "Pick a time in the future.",
    );
  });
});
