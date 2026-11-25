"use client";

import { useLayoutEffect, type ReactNode } from "react";

// Below lg, globals.css locks the document while this class is set, and the
// post table scrolls in the space that leaves. The editor and the other
// admin pages stay out of it. Hiding this route runs the cleanup, so leaving
// the list unlocks the page.
export function AdminListFrame({ children }: { children: ReactNode }) {
  useLayoutEffect(() => {
    document.documentElement.classList.add("admin-list");
    return () => document.documentElement.classList.remove("admin-list");
  }, []);
  return (
    <div className="max-lg:flex max-lg:min-h-0 max-lg:flex-1 max-lg:flex-col">
      {children}
    </div>
  );
}
