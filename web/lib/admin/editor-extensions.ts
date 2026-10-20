import Image from "@tiptap/extension-image";
import { TableKit } from "@tiptap/extension-table";
import { Markdown } from "@tiptap/markdown";
import StarterKit from "@tiptap/starter-kit";

// The editor loads and saves markdown, which goldmark renders on save. Every
// extension here needs a markdown form goldmark understands; underline has
// none, so it's off.
export function editorExtensions() {
  return [
    StarterKit.configure({
      underline: false,
      link: { openOnClick: false, defaultProtocol: "https" },
    }),
    Image,
    TableKit.configure({ table: { resizable: false } }),
    Markdown,
  ];
}
