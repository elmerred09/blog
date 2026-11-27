"use client";

import { useRouter } from "next/navigation";
import { useState, useTransition } from "react";

import { deletePost } from "@/app/admin/actions";
import { ConfirmDialog } from "@/components/admin/confirm-dialog";

type Props = {
  // slug is the stored one, whose public page the delete expires.
  post: { id: string; slug: string; title: string };
  // icon: the trash button on a post list row. text: the editor's button.
  variant: "icon" | "text";
  // list: go to the post list (from the editor, whose post is gone).
  // refresh: reload the current page (the list, without the post).
  after: "list" | "refresh";
};

const triggerClasses: Record<Props["variant"], string> = {
  icon: "flex size-10 cursor-pointer items-center justify-center rounded-lg border border-line bg-background text-muted hover:border-red-300 hover:text-red-700",
  text: "cursor-pointer rounded px-4 py-2 text-sm font-medium text-red-800 hover:bg-red-50 dark:text-red-300 dark:hover:bg-red-950",
};

// A delete button that asks in a modal first. The post is soft-deleted: it
// leaves the site and the admin, and its slug stays taken.
export function DeletePostButton({ post, variant, after }: Props) {
  const router = useRouter();
  const [open, setOpen] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [deleting, startDeleting] = useTransition();

  function confirm() {
    setError(null);
    startDeleting(async () => {
      let result: Awaited<ReturnType<typeof deletePost>>;
      try {
        result = await deletePost({ id: post.id, slug: post.slug });
      } catch {
        setError("Couldn't delete the post. Try again.");
        return;
      }
      if (!result.ok) {
        setError(
          result.error.code === "not_found"
            ? "This post was already deleted. Reload to see the current list."
            : result.error.message,
        );
        return;
      }
      setOpen(false);
      if (after === "list") {
        router.replace("/admin");
      }
      router.refresh();
    });
  }

  return (
    <>
      <button
        type="button"
        onClick={() => {
          setError(null);
          setOpen(true);
        }}
        aria-label={variant === "icon" ? `Delete ${post.title}` : undefined}
        title={variant === "icon" ? "Delete" : undefined}
        className={triggerClasses[variant]}
      >
        {variant === "icon" ? <TrashIcon /> : "Delete post"}
      </button>
      {open && (
        <ConfirmDialog
          title="Delete this post?"
          description={
            <>
              “{post.title}” leaves the site and the post list
              {after === "list" ? ", along with any unsaved edits" : ""}. You
              can&apos;t undo this from the admin.
            </>
          }
          confirmLabel="Delete post"
          pendingLabel="Deleting…"
          confirm={{ onConfirm: confirm }}
          cancelLabel="Cancel"
          onCancel={() => setOpen(false)}
          pending={deleting}
          error={error}
        />
      )}
    </>
  );
}

function TrashIcon() {
  return (
    <svg
      viewBox="0 0 24 24"
      fill="none"
      stroke="currentColor"
      strokeWidth="2"
      strokeLinecap="round"
      strokeLinejoin="round"
      aria-hidden="true"
      className="size-4"
    >
      <path d="M4 7h16M10 11v6M14 11v6" />
      <path d="M6 7l1 12a2 2 0 0 0 2 2h6a2 2 0 0 0 2-2l1-12" />
      <path d="M9 7V4h6v3" />
    </svg>
  );
}
