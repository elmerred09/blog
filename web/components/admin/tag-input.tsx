"use client";

import { useId, useState, type KeyboardEvent } from "react";

import { slugify } from "@/lib/admin/slug";

type Props = {
  value: string[];
  onChange: (tags: string[]) => void;
  // Every existing tag's display name, for autocomplete. New tags are allowed.
  suggestions: string[];
  error?: string;
};

// Two names with the same slug are the same tag to the API, which keeps the
// first. Comparing slugs here stops the duplicate before it's sent.
function sameTag(name: string): string {
  return slugify(name) || name.trim().toLowerCase();
}

export function TagInput({ value, onChange, suggestions, error }: Props) {
  const [text, setText] = useState("");
  const id = useId();
  const taken = new Set(value.map(sameTag));

  function add(names: string[]) {
    const added: string[] = [];
    const seen = new Set(taken);
    for (const raw of names) {
      const name = raw.trim();
      if (name !== "" && !seen.has(sameTag(name))) {
        seen.add(sameTag(name));
        added.push(name);
      }
    }
    if (added.length > 0) {
      onChange([...value, ...added]);
    }
  }

  function commit() {
    add([text]);
    setText("");
  }

  // A comma can arrive without a keydown: pasted text, or a mobile keyboard
  // that reports every key as "Unidentified". Everything before the last
  // comma becomes tags.
  function onText(next: string) {
    const parts = next.split(",");
    const rest = parts.pop() ?? "";
    add(parts);
    setText(rest);
  }

  function onKeyDown(event: KeyboardEvent<HTMLInputElement>) {
    if (event.key === "Enter" || event.key === ",") {
      event.preventDefault();
      commit();
    } else if (event.key === "Backspace" && text === "" && value.length > 0) {
      onChange(value.slice(0, -1));
    }
  }

  return (
    <div className="space-y-1">
      <label htmlFor={id} className="block text-sm font-medium">
        Tags
      </label>
      <div className="flex flex-wrap items-center gap-2 rounded border border-zinc-300 px-2 py-1.5 focus-within:border-zinc-500 dark:border-zinc-700">
        {value.length > 0 && (
          <ul className="contents">
            {value.map((tag) => (
              <li
                key={tag}
                className="flex items-center gap-1 rounded bg-zinc-100 px-1.5 py-0.5 text-sm dark:bg-zinc-800"
              >
                <span>{tag}</span>
                <button
                  type="button"
                  aria-label={`Remove ${tag}`}
                  onClick={() => onChange(value.filter((t) => t !== tag))}
                  className="text-zinc-500 hover:text-zinc-900 dark:hover:text-zinc-100"
                >
                  ×
                </button>
              </li>
            ))}
          </ul>
        )}
        <input
          id={id}
          list={`${id}-suggestions`}
          value={text}
          onChange={(event) => onText(event.target.value)}
          onKeyDown={onKeyDown}
          onBlur={commit}
          aria-invalid={error ? true : undefined}
          aria-describedby={error ? `${id}-error` : undefined}
          placeholder={value.length === 0 ? "Add a tag, then Enter" : ""}
          className="min-w-32 flex-1 bg-transparent py-0.5 text-sm outline-none"
        />
        <datalist id={`${id}-suggestions`}>
          {suggestions
            .filter((name) => !taken.has(sameTag(name)))
            .map((name) => (
              <option key={name} value={name} />
            ))}
        </datalist>
      </div>
      {error && (
        <p
          id={`${id}-error`}
          className="text-sm text-red-700 dark:text-red-400"
        >
          {error}
        </p>
      )}
    </div>
  );
}
