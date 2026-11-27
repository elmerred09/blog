"use client";

import Link from "next/link";
import { useId, useLayoutEffect, useRef, type ReactNode } from "react";

type Props = {
  title: string;
  description: ReactNode;
  cancelLabel: string;
  onCancel: () => void;
  confirmLabel: string;
  // The confirm control is a link when it only navigates (discarding edits),
  // or a button that runs onConfirm (deleting).
  confirm: { href: string } | { onConfirm: () => void };
  // While the confirmed action runs, the dialog can't be dismissed and the
  // confirm button shows pendingLabel.
  pending?: boolean;
  pendingLabel?: string;
  error?: string | null;
};

const confirmClasses =
  "cursor-pointer rounded px-4 py-2 text-sm font-medium text-red-800 hover:bg-red-50 disabled:cursor-wait disabled:opacity-60 dark:text-red-300 dark:hover:bg-red-950";

// A modal that asks before a destructive step. The safe choice (cancelLabel)
// has focus when it opens; Escape and a click on the backdrop pick it too.
export function ConfirmDialog({
  title,
  description,
  cancelLabel,
  onCancel,
  confirmLabel,
  confirm,
  pending = false,
  pendingLabel,
  error,
}: Props) {
  const ref = useRef<HTMLDialogElement>(null);
  const cancel = useRef<HTMLButtonElement>(null);
  const titleId = useId();
  const bodyId = useId();

  useLayoutEffect(() => {
    const dialog = ref.current;
    if (!dialog) {
      return;
    }
    dialog.showModal();
    // showModal focuses the first control, the confirm one. The safe choice
    // should have focus: the destructive one is what to confirm.
    cancel.current?.focus();
    return () => {
      if (dialog.open) {
        dialog.close();
      }
    };
  }, []);

  function dismiss() {
    if (!pending) {
      onCancel();
    }
  }

  return (
    <dialog
      ref={ref}
      aria-labelledby={titleId}
      aria-describedby={bodyId}
      onCancel={(event) => {
        event.preventDefault();
        dismiss();
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
          dismiss();
        }
      }}
      className="fixed top-1/2 left-1/2 w-[min(28rem,calc(100vw-2rem))] -translate-x-1/2 -translate-y-1/2 rounded border border-zinc-300 bg-background p-6 text-left text-foreground dark:border-zinc-700"
    >
      <h2 id={titleId} className="text-lg font-semibold tracking-tight">
        {title}
      </h2>
      <p id={bodyId} className="mt-2 text-sm text-muted">
        {description}
      </p>
      {error ? (
        <p role="alert" className="mt-3 text-sm text-red-800">
          {error}
        </p>
      ) : null}
      <div className="mt-6 flex flex-wrap justify-end gap-2">
        {"href" in confirm ? (
          <Link href={confirm.href} className={confirmClasses}>
            {confirmLabel}
          </Link>
        ) : (
          <button
            type="button"
            onClick={confirm.onConfirm}
            disabled={pending}
            className={confirmClasses}
          >
            {pending && pendingLabel ? pendingLabel : confirmLabel}
          </button>
        )}
        <button
          ref={cancel}
          type="button"
          onClick={dismiss}
          disabled={pending}
          className="cursor-pointer rounded bg-zinc-900 px-4 py-2 text-sm font-medium text-white hover:bg-zinc-700 disabled:cursor-wait disabled:opacity-60 dark:bg-zinc-100 dark:text-zinc-900 dark:hover:bg-zinc-300"
        >
          {cancelLabel}
        </button>
      </div>
    </dialog>
  );
}
