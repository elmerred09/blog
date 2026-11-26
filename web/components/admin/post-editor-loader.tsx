"use client";

import dynamic from "next/dynamic";

import { PostEditorSkeleton } from "@/components/skeleton";

// The editor renders in the browser only. Tiptap can't render on the server,
// and the publish controls show the author's local time; server HTML for
// either would only mismatch on hydration.
export const PostEditorLoader = dynamic(
  () => import("./post-editor").then((m) => m.PostEditor),
  {
    ssr: false,
    loading: () => <PostEditorSkeleton />,
  },
);
