"use client";

import { useId } from "react";

import type { PublishMode, PublishState } from "@/lib/admin/publish";
import { formatDateTime } from "@/lib/format";

type Props = {
  value: PublishState;
  onChange: (state: PublishState) => void;
  // The post's saved published_at, or null for a draft or a new post.
  publishedAt: string | null;
  now?: Date;
  error?: string;
};

// The editor renders only in the browser, so the local time zone here is
// the author's, not the server's.
export function PublishControls({
  value,
  onChange,
  publishedAt,
  now = new Date(),
  error,
}: Props) {
  const id = useId();
  const live = publishedAt !== null && new Date(publishedAt) <= now;
  const zone = Intl.DateTimeFormat().resolvedOptions().timeZone;

  const modes: [PublishMode, string][] = [
    ["draft", "Draft"],
    ["now", live ? `Published ${formatDateTime(publishedAt)}` : "Publish now"],
    ["schedule", "Schedule"],
  ];

  return (
    <fieldset className="space-y-2">
      <legend className="text-sm font-medium">Publishing</legend>
      <div className="flex flex-wrap gap-x-5 gap-y-1 text-sm">
        {modes.map(([mode, label]) => (
          <label key={mode} className="flex items-center gap-1.5">
            <input
              type="radio"
              name={`${id}-mode`}
              value={mode}
              checked={value.mode === mode}
              onChange={() => onChange({ ...value, mode })}
            />
            {label}
          </label>
        ))}
      </div>
      {value.mode === "schedule" && (
        <label className="block space-y-1 text-sm">
          <span>Publish at ({zone})</span>
          <input
            type="datetime-local"
            value={value.scheduleLocal}
            onChange={(event) =>
              onChange({ ...value, scheduleLocal: event.target.value })
            }
            className="block rounded border border-zinc-300 px-2 py-1 dark:border-zinc-700 dark:bg-zinc-900"
          />
        </label>
      )}
      {error && (
        <p role="alert" className="text-sm text-red-700 dark:text-red-400">
          {error}
        </p>
      )}
    </fieldset>
  );
}
