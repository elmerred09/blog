"use client";

import { useRouter } from "next/navigation";
import {
  useId,
  useLayoutEffect,
  useRef,
  useState,
  useTransition,
  type ReactNode,
} from "react";

import { createImageUpload, previewPost, savePost } from "@/app/admin/actions";
import { Editor } from "@/components/admin/editor";
import { PublishControls } from "@/components/admin/publish-controls";
import { TagInput } from "@/components/admin/tag-input";
import { PostBody } from "@/components/post-body";
import {
  initialPublishState,
  publishedAtFor,
  type PublishState,
} from "@/lib/admin/publish";
import { slugify } from "@/lib/admin/slug";
import { uploadImage } from "@/lib/admin/upload";
import type { Failure } from "@/lib/api/admin";
import type { AdminPost } from "@/lib/api/types";

type Props = {
  // null for a new post.
  post: AdminPost | null;
  // Every existing tag's name, for autocomplete.
  tagSuggestions: string[];
};

// What the last load or save returned. updatedAt is the lock token: sent back
// on the next save exactly as received, never parsed.
type Saved = {
  id: string;
  slug: string;
  updatedAt: string;
  publishedAt: string | null;
  state: AdminPost["state"];
};

type Problems = Partial<Record<string, string>>;

function upload(file: File) {
  return uploadImage(file, createImageUpload);
}

function savedFrom(post: AdminPost): Saved {
  return {
    id: post.id,
    slug: post.slug,
    updatedAt: post.updated_at,
    publishedAt: post.published_at,
    state: post.state,
  };
}

// Groups the API's problems by field; a field can appear more than once.
function byField(problems: Failure["problems"]): Problems {
  const grouped: Problems = {};
  for (const { field, message } of problems) {
    grouped[field] = grouped[field] ? `${grouped[field]}; ${message}` : message;
  }
  return grouped;
}

function failureMessage(error: Failure): {
  problems: Problems;
  alert?: string;
} {
  switch (error.code) {
    case "invalid_request":
      return error.problems.length > 0
        ? { problems: byField(error.problems) }
        : { problems: {}, alert: error.message };
    case "slug_taken":
      return { problems: { slug: "Another post already uses this slug." } };
    case "conflict":
      return {
        problems: {},
        alert:
          "This post changed since you opened it, maybe in another tab. Copy your changes, reload, and apply them again.",
      };
    case "not_found":
      return { problems: {}, alert: "This post no longer exists." };
    case "too_large":
      return { problems: { body_md: error.message } };
  }
}

// Next keeps a visited route mounted but hidden (<Activity>) after navigating
// away. A new post's form that has created its post would come back bound to
// it, so once hidden it starts over. An unsaved draft is kept.
export function PostEditor(props: Props) {
  const [generation, setGeneration] = useState(0);
  const created = useRef(false);
  useLayoutEffect(
    () => () => {
      if (created.current) {
        created.current = false;
        setGeneration((g) => g + 1);
      }
    },
    [],
  );
  return (
    <PostForm
      key={generation}
      {...props}
      onCreated={() => {
        created.current = true;
      }}
    />
  );
}

function PostForm({
  post,
  tagSuggestions,
  onCreated,
}: Props & { onCreated: () => void }) {
  const router = useRouter();
  const [saved, setSaved] = useState<Saved | null>(post && savedFrom(post));
  const [title, setTitle] = useState(post?.title ?? "");
  const [slug, setSlug] = useState(post?.slug ?? "");
  // A new post's slug follows its title until edited by hand.
  const [slugEdited, setSlugEdited] = useState(post !== null);
  const [summary, setSummary] = useState(post?.summary ?? "");
  const [initialBody] = useState(post?.body_md ?? "");
  const body = useRef(initialBody);
  const [tags, setTags] = useState(post?.tags.map((t) => t.name) ?? []);
  const [publish, setPublish] = useState<PublishState>(() =>
    initialPublishState(post?.published_at ?? null, new Date()),
  );
  const [problems, setProblems] = useState<Problems>({});
  const [alert, setAlert] = useState<string | null>(null);
  const [status, setStatus] = useState<string | null>(null);
  const [saving, startSaving] = useTransition();
  const [tab, setTab] = useState<"write" | "preview">("write");
  const [preview, setPreview] = useState<
    { html: string } | { error: string } | null
  >(null);
  const [previewing, startPreview] = useTransition();

  function onTitle(value: string) {
    setTitle(value);
    if (!slugEdited) {
      setSlug(slugify(value));
    }
  }

  function save() {
    setStatus(null);
    const publishedAt = publishedAtFor(
      publish,
      saved?.publishedAt ?? null,
      new Date(),
    );
    if (!publishedAt.ok) {
      setProblems({ published_at: publishedAt.message });
      return;
    }
    startSaving(async () => {
      const result = await savePost({
        id: saved?.id ?? null,
        loadedSlug: saved?.slug ?? null,
        updatedAt: saved?.updatedAt ?? null,
        post: {
          slug,
          title,
          summary,
          body_md: body.current,
          published_at: publishedAt.value,
          tags,
        },
      });
      if (!result.ok) {
        const { problems, alert } = failureMessage(result.error);
        setProblems(problems);
        setAlert(alert ?? null);
        return;
      }
      setProblems({});
      setAlert(null);
      const next = savedFrom(result.value);
      setSaved(next);
      setSlug(next.slug);
      setSlugEdited(true);
      setPublish(initialPublishState(next.publishedAt, new Date()));
      setStatus(`Saved (${next.state}).`);
      if (saved === null) {
        onCreated();
        router.replace(`/admin/posts/${next.id}`);
      }
    });
  }

  function showPreview() {
    setTab("preview");
    setPreview(null);
    startPreview(async () => {
      const result = await previewPost(body.current);
      setPreview(
        result.ok
          ? { html: result.value }
          : {
              error: result.error.problems[0]?.message ?? result.error.message,
            },
      );
    });
  }

  const renaming =
    saved !== null && saved.state === "published" && slug !== saved.slug;

  return (
    <form
      onSubmit={(event) => {
        event.preventDefault();
        save();
      }}
      className="space-y-6"
    >
      <TextField
        label="Title"
        value={title}
        onChange={onTitle}
        error={problems.title}
      />
      <TextField
        label="Slug"
        value={slug}
        onChange={(value) => {
          setSlug(value);
          setSlugEdited(true);
        }}
        error={problems.slug}
        hint={
          renaming
            ? `This post is published: changing its slug breaks links to /posts/${saved.slug}.`
            : undefined
        }
      />
      <TextField
        label="Summary"
        value={summary}
        onChange={setSummary}
        error={problems.summary}
      />

      <div className="space-y-2">
        <div role="tablist" className="flex gap-4 text-sm">
          <Tab selected={tab === "write"} onClick={() => setTab("write")}>
            Write
          </Tab>
          <Tab selected={tab === "preview"} onClick={showPreview}>
            Preview
          </Tab>
        </div>
        {/* Hidden, not unmounted: the editor holds the undo history. */}
        <div hidden={tab !== "write"}>
          <Editor
            initialMarkdown={initialBody}
            onChange={(markdown) => {
              body.current = markdown;
            }}
            upload={upload}
          />
        </div>
        {tab === "preview" && (
          <div className="rounded border border-zinc-200 px-6 py-5 dark:border-zinc-800">
            {previewing || preview === null ? (
              <p className="text-zinc-500">Rendering…</p>
            ) : "error" in preview ? (
              <p role="alert" className="text-red-700 dark:text-red-400">
                {preview.error}
              </p>
            ) : (
              <article>
                <h1 className="mb-8 text-3xl font-bold tracking-tight">
                  {title}
                </h1>
                <PostBody html={preview.html} />
              </article>
            )}
          </div>
        )}
        {problems.body_md && (
          <p className="text-sm text-red-700 dark:text-red-400">
            {problems.body_md}
          </p>
        )}
      </div>

      <TagInput
        value={tags}
        onChange={setTags}
        suggestions={tagSuggestions}
        error={problems.tags}
      />
      <PublishControls
        value={publish}
        onChange={setPublish}
        publishedAt={saved?.publishedAt ?? null}
        error={problems.published_at}
      />

      {alert && (
        <p
          role="alert"
          className="rounded border border-red-200 bg-red-50 px-3 py-2 text-sm text-red-800 dark:border-red-900 dark:bg-red-950 dark:text-red-300"
        >
          {alert}
        </p>
      )}
      <div className="flex items-center gap-4">
        <button
          type="submit"
          disabled={saving}
          className="rounded bg-zinc-900 px-4 py-2 text-sm font-medium text-white hover:bg-zinc-700 disabled:opacity-50 dark:bg-zinc-100 dark:text-zinc-900 dark:hover:bg-zinc-300"
        >
          {saving ? "Saving…" : "Save"}
        </button>
        {status && (
          <p role="status" className="text-sm text-zinc-600 dark:text-zinc-400">
            {status}
          </p>
        )}
      </div>
    </form>
  );
}

function TextField({
  label,
  value,
  onChange,
  error,
  hint,
}: {
  label: string;
  value: string;
  onChange: (value: string) => void;
  error?: string;
  hint?: string;
}) {
  const id = useId();
  return (
    <div className="space-y-1">
      <label htmlFor={id} className="block text-sm font-medium">
        {label}
      </label>
      <input
        id={id}
        value={value}
        onChange={(event) => onChange(event.target.value)}
        aria-invalid={error ? true : undefined}
        aria-describedby={error || hint ? `${id}-note` : undefined}
        className="block w-full rounded border border-zinc-300 px-3 py-2 aria-invalid:border-red-600 dark:border-zinc-700 dark:bg-zinc-900"
      />
      {(error || hint) && (
        <p
          id={`${id}-note`}
          className={`text-sm ${error ? "text-red-700 dark:text-red-400" : "text-amber-700 dark:text-amber-400"}`}
        >
          {error ?? hint}
        </p>
      )}
    </div>
  );
}

function Tab({
  selected,
  onClick,
  children,
}: {
  selected: boolean;
  onClick: () => void;
  children: ReactNode;
}) {
  return (
    <button
      type="button"
      role="tab"
      aria-selected={selected}
      onClick={onClick}
      className="border-b-2 border-transparent pb-1 font-medium text-zinc-500 aria-selected:border-zinc-900 aria-selected:text-zinc-900 dark:aria-selected:border-zinc-100 dark:aria-selected:text-zinc-100"
    >
      {children}
    </button>
  );
}
