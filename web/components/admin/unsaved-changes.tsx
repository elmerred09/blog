"use client";

import Link from "next/link";
import {
  createContext,
  useContext,
  useEffect,
  useId,
  useLayoutEffect,
  useRef,
  useState,
  type ReactNode,
} from "react";

type Guard = {
  // True when the click should stay on the page. Opens the dialog.
  block: () => boolean;
  register: (isDirty: () => boolean) => () => void;
};

const UnsavedContext = createContext<Guard | null>(null);

export function useUnsavedGuard(): Guard | null {
  return useContext(UnsavedContext);
}

// Wraps an editor page so its Back link can ask before leaving. Only that
// link is guarded: browser Back, refresh, and closing the tab are separate.
export function UnsavedChanges({ children }: { children: ReactNode }) {
  const isDirty = useRef<() => boolean>(() => false);
  const [open, setOpen] = useState(false);
  const [guard] = useState<Guard>(() => ({
    block: () => {
      if (!isDirty.current()) {
        return false;
      }
      setOpen(true);
      return true;
    },
    register: (fn) => {
      isDirty.current = fn;
      return () => {
        if (isDirty.current === fn) {
          isDirty.current = () => false;
        }
      };
    },
  }));
  // Next keeps this page mounted while it is hidden. A modal dialog stays in
  // the top layer across that, so close it before the page is shown again.
  useLayoutEffect(() => {
    return () => setOpen(false);
  }, []);

  return (
    <UnsavedContext.Provider value={guard}>
      {children}
      {open && <DiscardDialog onStay={() => setOpen(false)} />}
    </UnsavedContext.Provider>
  );
}

// No-op without a provider, so the editor can render on its own in tests.
export function useRegisterUnsaved(isDirty: () => boolean) {
  const guard = useContext(UnsavedContext);
  const latest = useRef(isDirty);
  useEffect(() => {
    latest.current = isDirty;
  });
  useEffect(() => {
    if (guard === null) {
      return;
    }
    return guard.register(() => latest.current());
  }, [guard]);
}

function DiscardDialog({ onStay }: { onStay: () => void }) {
  const ref = useRef<HTMLDialogElement>(null);
  const keepEditing = useRef<HTMLButtonElement>(null);
  const titleId = useId();
  const bodyId = useId();

  useLayoutEffect(() => {
    const dialog = ref.current;
    if (!dialog) {
      return;
    }
    dialog.showModal();
    // showModal focuses the first control, which is Discard. Keep editing is
    // the one that should have focus: leaving is the choice to confirm.
    keepEditing.current?.focus();
    return () => {
      if (dialog.open) {
        dialog.close();
      }
    };
  }, []);

  return (
    <dialog
      ref={ref}
      aria-labelledby={titleId}
      aria-describedby={bodyId}
      onCancel={(event) => {
        event.preventDefault();
        onStay();
      }}
      onClick={(event) => {
        if (event.target !== event.currentTarget) {
          return;
        }
        const rect = event.currentTarget.getBoundingClientRect();
        const inside =
          event.clientX >= rect.left &&
          event.clientX <= rect.right &&
          event.clientY >= rect.top &&
          event.clientY <= rect.bottom;
        if (!inside) {
          onStay();
        }
      }}
      className="fixed top-1/2 left-1/2 w-[min(28rem,calc(100vw-2rem))] -translate-x-1/2 -translate-y-1/2 rounded border border-zinc-300 bg-background p-6 text-left text-foreground dark:border-zinc-700"
    >
      <h2 id={titleId} className="text-lg font-semibold tracking-tight">
        Discard unsaved changes?
      </h2>
      <p id={bodyId} className="mt-2 text-sm text-muted">
        This post has unsaved edits. Leaving goes back to the post list.
      </p>
      <div className="mt-6 flex flex-wrap justify-end gap-2">
        <Link
          href="/admin"
          className="rounded px-4 py-2 text-sm font-medium text-red-800 hover:bg-red-50 dark:text-red-300 dark:hover:bg-red-950"
        >
          Discard changes
        </Link>
        <button
          ref={keepEditing}
          type="button"
          onClick={onStay}
          className="cursor-pointer rounded bg-zinc-900 px-4 py-2 text-sm font-medium text-white hover:bg-zinc-700 dark:bg-zinc-100 dark:text-zinc-900 dark:hover:bg-zinc-300"
        >
          Keep editing
        </button>
      </div>
    </dialog>
  );
}
