"use client";

import type { Editor as TiptapEditor } from "@tiptap/core";
import { EditorContent, useEditor, useEditorState } from "@tiptap/react";
import { useRef, useState, type ReactNode } from "react";

import { editorExtensions } from "@/lib/admin/editor-extensions";
import { altFromFileName, type UploadResult } from "@/lib/admin/upload";

type Props = {
  initialMarkdown: string;
  // Called with the whole document as markdown after every change.
  onChange: (markdown: string) => void;
  upload: (file: File) => Promise<UploadResult>;
};

// The types internal/media accepts; the API checks them again before signing.
const IMAGE_TYPES = "image/png,image/jpeg,image/webp,image/gif";

function imageFiles(data: DataTransfer | null): File[] {
  return [...(data?.files ?? [])].filter((f) => f.type.startsWith("image/"));
}

export function Editor({ initialMarkdown, onChange, upload }: Props) {
  const [uploads, setUploads] = useState<string[]>([]);
  const [failures, setFailures] = useState<string[]>([]);
  // editorProps closures are created once, before the editor exists.
  const editorRef = useRef<TiptapEditor | null>(null);

  async function insertImages(files: File[], pos?: number) {
    for (const file of files) {
      setUploads((u) => [...u, file.name]);
      const result = await upload(file);
      setUploads((u) => u.filter((name) => name !== file.name));
      const editor = editorRef.current;
      if (!result.ok) {
        setFailures((f) => [...f, `${file.name}: ${result.message}`]);
        continue;
      }
      if (!editor || editor.isDestroyed) {
        return;
      }
      const image = {
        type: "image",
        attrs: { src: result.url, alt: altFromFileName(file.name) },
      };
      const chain = editor.chain().focus();
      if (pos === undefined) {
        chain.insertContent(image).run();
      } else {
        chain
          .insertContentAt(Math.min(pos, editor.state.doc.content.size), image)
          .run();
      }
    }
  }

  const editor = useEditor({
    extensions: editorExtensions(),
    content: initialMarkdown,
    contentType: "markdown",
    // Rendered on the client only; see post-editor-loader.tsx.
    immediatelyRender: false,
    editorProps: {
      attributes: {
        role: "textbox",
        "aria-multiline": "true",
        "aria-label": "Body",
        class:
          "prose max-w-none prose-zinc dark:prose-invert min-h-96 px-4 py-3 focus:outline-none",
      },
      handlePaste: (_view, event) => {
        const files = imageFiles(event.clipboardData);
        if (files.length === 0) {
          return false;
        }
        void insertImages(files);
        return true;
      },
      handleDrop: (view, event, _slice, moved) => {
        const files = moved ? [] : imageFiles(event.dataTransfer);
        if (files.length === 0) {
          return false;
        }
        event.preventDefault();
        const at = view.posAtCoords({
          left: event.clientX,
          top: event.clientY,
        });
        void insertImages(files, at?.pos);
        return true;
      },
    },
    onCreate: ({ editor }) => {
      editorRef.current = editor;
    },
    onUpdate: ({ editor }) => onChange(editor.getMarkdown()),
  });

  if (!editor) {
    return <EditorFrame toolbar={null}>{null}</EditorFrame>;
  }

  return (
    <EditorFrame
      toolbar={
        <Toolbar
          editor={editor}
          onImages={(files) => void insertImages(files)}
        />
      }
    >
      <EditorContent editor={editor} />
      {(uploads.length > 0 || failures.length > 0) && (
        <div className="space-y-1 border-t border-zinc-200 px-4 py-2 text-sm dark:border-zinc-800">
          {uploads.map((name) => (
            <p key={name} className="text-zinc-500">
              Uploading {name}…
            </p>
          ))}
          {failures.length > 0 && (
            <div role="alert" className="text-red-700 dark:text-red-400">
              {failures.map((message, i) => (
                <p key={i}>Upload failed. {message}</p>
              ))}
              <button
                type="button"
                onClick={() => setFailures([])}
                className="underline"
              >
                Dismiss
              </button>
            </div>
          )}
        </div>
      )}
    </EditorFrame>
  );
}

function EditorFrame({
  toolbar,
  children,
}: {
  toolbar: ReactNode;
  children: ReactNode;
}) {
  return (
    <div className="rounded border border-zinc-300 dark:border-zinc-700">
      <div className="flex min-h-10 flex-wrap items-center gap-1 border-b border-zinc-200 px-2 py-1 dark:border-zinc-800">
        {toolbar}
      </div>
      {children}
    </div>
  );
}

function Toolbar({
  editor,
  onImages,
}: {
  editor: TiptapEditor;
  onImages: (files: File[]) => void;
}) {
  const fileInput = useRef<HTMLInputElement>(null);
  const state = useEditorState({
    editor,
    selector: ({ editor: e }) => ({
      bold: e.isActive("bold"),
      italic: e.isActive("italic"),
      strike: e.isActive("strike"),
      code: e.isActive("code"),
      h2: e.isActive("heading", { level: 2 }),
      h3: e.isActive("heading", { level: 3 }),
      bulletList: e.isActive("bulletList"),
      orderedList: e.isActive("orderedList"),
      blockquote: e.isActive("blockquote"),
      link: e.isActive("link"),
      codeBlock: e.isActive("codeBlock"),
      language: String(e.getAttributes("codeBlock").language ?? ""),
      table: e.isActive("table"),
      canUndo: e.can().undo(),
      canRedo: e.can().redo(),
    }),
  });

  const chain = () => editor.chain().focus();

  function editLink() {
    const current = String(editor.getAttributes("link").href ?? "");
    const href = window.prompt("Link URL (empty to remove)", current);
    if (href === null) {
      return;
    }
    if (href.trim() === "") {
      chain().extendMarkRange("link").unsetLink().run();
    } else {
      chain().extendMarkRange("link").setLink({ href: href.trim() }).run();
    }
  }

  return (
    <>
      <Button
        label="Bold"
        shortcut="Ctrl+B"
        pressed={state.bold}
        onClick={() => chain().toggleBold().run()}
      >
        <b>B</b>
      </Button>
      <Button
        label="Italic"
        shortcut="Ctrl+I"
        pressed={state.italic}
        onClick={() => chain().toggleItalic().run()}
      >
        <i>I</i>
      </Button>
      <Button
        label="Strikethrough"
        shortcut="Ctrl+Shift+S"
        pressed={state.strike}
        onClick={() => chain().toggleStrike().run()}
      >
        <s>S</s>
      </Button>
      <Button
        label="Inline code"
        shortcut="Ctrl+E"
        pressed={state.code}
        onClick={() => chain().toggleCode().run()}
      >
        <code>{"`c`"}</code>
      </Button>
      <Separator />
      <Button
        label="Heading 2"
        shortcut="Ctrl+Alt+2"
        pressed={state.h2}
        onClick={() => chain().toggleHeading({ level: 2 }).run()}
      >
        H2
      </Button>
      <Button
        label="Heading 3"
        shortcut="Ctrl+Alt+3"
        pressed={state.h3}
        onClick={() => chain().toggleHeading({ level: 3 }).run()}
      >
        H3
      </Button>
      <Button
        label="Bullet list"
        shortcut="Ctrl+Shift+8"
        pressed={state.bulletList}
        onClick={() => chain().toggleBulletList().run()}
      >
        •
      </Button>
      <Button
        label="Numbered list"
        shortcut="Ctrl+Shift+7"
        pressed={state.orderedList}
        onClick={() => chain().toggleOrderedList().run()}
      >
        1.
      </Button>
      <Button
        label="Quote"
        shortcut="Ctrl+Shift+B"
        pressed={state.blockquote}
        onClick={() => chain().toggleBlockquote().run()}
      >
        ❝
      </Button>
      <Separator />
      <Button label="Link" pressed={state.link} onClick={editLink}>
        Link
      </Button>
      <Button
        label="Code block"
        shortcut="Ctrl+Alt+C"
        pressed={state.codeBlock}
        onClick={() => chain().toggleCodeBlock().run()}
      >
        {"{ }"}
      </Button>
      {state.codeBlock && (
        <input
          aria-label="Code language"
          placeholder="language"
          value={state.language}
          onChange={(event) =>
            editor
              .chain()
              .updateAttributes("codeBlock", {
                language: event.target.value.trim() || null,
              })
              .run()
          }
          className="w-24 rounded border border-zinc-300 px-1.5 py-0.5 text-sm dark:border-zinc-700 dark:bg-zinc-900"
        />
      )}
      <Button
        label="Table"
        pressed={state.table}
        onClick={() =>
          chain().insertTable({ rows: 3, cols: 3, withHeaderRow: true }).run()
        }
      >
        Table
      </Button>
      {state.table && (
        <>
          <Button label="Add row" onClick={() => chain().addRowAfter().run()}>
            +Row
          </Button>
          <Button
            label="Add column"
            onClick={() => chain().addColumnAfter().run()}
          >
            +Col
          </Button>
          <Button label="Delete row" onClick={() => chain().deleteRow().run()}>
            −Row
          </Button>
          <Button
            label="Delete column"
            onClick={() => chain().deleteColumn().run()}
          >
            −Col
          </Button>
          <Button
            label="Delete table"
            onClick={() => chain().deleteTable().run()}
          >
            ×Table
          </Button>
        </>
      )}
      <Button label="Image" onClick={() => fileInput.current?.click()}>
        Image
      </Button>
      <input
        ref={fileInput}
        type="file"
        accept={IMAGE_TYPES}
        multiple
        hidden
        aria-label="Upload image"
        onChange={(event) => {
          const files = [...(event.target.files ?? [])];
          event.target.value = "";
          if (files.length > 0) {
            onImages(files);
          }
        }}
      />
      <Separator />
      <Button
        label="Undo"
        shortcut="Ctrl+Z"
        disabled={!state.canUndo}
        onClick={() => chain().undo().run()}
      >
        ↶
      </Button>
      <Button
        label="Redo"
        shortcut="Ctrl+Shift+Z"
        disabled={!state.canRedo}
        onClick={() => chain().redo().run()}
      >
        ↷
      </Button>
    </>
  );
}

function Button({
  label,
  shortcut,
  pressed,
  disabled,
  onClick,
  children,
}: {
  label: string;
  shortcut?: string;
  pressed?: boolean;
  disabled?: boolean;
  onClick: () => void;
  children: ReactNode;
}) {
  return (
    <button
      type="button"
      aria-label={label}
      aria-pressed={pressed}
      title={shortcut ? `${label} (${shortcut})` : label}
      disabled={disabled}
      onClick={onClick}
      className="min-w-8 rounded px-1.5 py-1 text-sm text-zinc-700 hover:bg-zinc-100 disabled:opacity-40 aria-pressed:bg-zinc-200 dark:text-zinc-300 dark:hover:bg-zinc-800 dark:aria-pressed:bg-zinc-700"
    >
      {children}
    </button>
  );
}

function Separator() {
  return (
    <span aria-hidden className="mx-1 h-5 w-px bg-zinc-200 dark:bg-zinc-700" />
  );
}
