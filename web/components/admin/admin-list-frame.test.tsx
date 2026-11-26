import { render } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { AdminListFrame } from "@/components/admin/admin-list-frame";

describe("AdminListFrame", () => {
  it("locks page scroll while the list is showing", () => {
    const view = render(
      <AdminListFrame>
        <p>Posts</p>
      </AdminListFrame>,
    );

    expect(document.documentElement.classList.contains("admin-list")).toBe(
      true,
    );

    view.unmount();

    expect(document.documentElement.classList.contains("admin-list")).toBe(
      false,
    );
  });
});
