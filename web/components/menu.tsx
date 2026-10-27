"use client";

import Link from "next/link";
import { useEffect, useId, useRef, useState } from "react";

const items = [
  { href: "/", label: "Home" },
  { href: "/search", label: "Search" },
];

export function Menu() {
  const [open, setOpen] = useState(false);
  const rootRef = useRef<HTMLDivElement>(null);
  const menuId = useId();

  useEffect(() => {
    if (!open) {
      return;
    }
    function onPointerDown(event: PointerEvent) {
      if (!rootRef.current?.contains(event.target as Node)) {
        setOpen(false);
      }
    }
    function onKeyDown(event: KeyboardEvent) {
      if (event.key === "Escape") {
        setOpen(false);
      }
    }
    document.addEventListener("pointerdown", onPointerDown);
    document.addEventListener("keydown", onKeyDown);
    return () => {
      document.removeEventListener("pointerdown", onPointerDown);
      document.removeEventListener("keydown", onKeyDown);
    };
  }, [open]);

  return (
    <div ref={rootRef} className="relative">
      <button
        type="button"
        aria-expanded={open}
        aria-controls={menuId}
        onClick={() => setOpen((value) => !value)}
        className="h-10 cursor-pointer rounded-lg border border-line bg-card px-4 text-sm hover:border-line-strong"
      >
        {open ? "Close" : "Menu"}
      </button>
      {open && (
        <ul
          id={menuId}
          className="absolute right-0 z-20 mt-2 w-32 space-y-2 rounded-lg border border-line bg-card p-5"
        >
          {items.map((item) => (
            <li key={item.href}>
              <Link
                href={item.href}
                onClick={() => setOpen(false)}
                className="block cursor-pointer border-b border-line pb-2 text-sm transition-colors hover:border-black/40"
              >
                {item.label}
              </Link>
            </li>
          ))}
        </ul>
      )}
    </div>
  );
}
