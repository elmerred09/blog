"use client";

import {
  createContext,
  useContext,
  useEffect,
  useLayoutEffect,
  useRef,
  useState,
  type ReactNode,
} from "react";

import { ConfirmDialog } from "@/components/admin/confirm-dialog";

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
  return (
    <ConfirmDialog
      title="Discard unsaved changes?"
      description="This post has unsaved edits. Leaving goes back to the post list."
      confirmLabel="Discard changes"
      confirm={{ href: "/admin" }}
      cancelLabel="Keep editing"
      onCancel={onStay}
    />
  );
}
