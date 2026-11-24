"use client";

import { useSyncExternalStore } from "react";

import { formatCompactDateTime, formatZonedDateTime } from "@/lib/format";

const subscribe = () => () => {};

// A time in the reader's own zone. The server doesn't know that zone, so the
// text appears only once the page runs in the browser; rendering it on the
// server would mismatch on hydration.
export function LocalTime({ iso }: { iso: string }) {
  const inBrowser = useSyncExternalStore(
    subscribe,
    () => true,
    () => false,
  );
  if (!inBrowser) {
    return <time dateTime={iso} />;
  }
  return (
    <time dateTime={iso} title={formatZonedDateTime(iso)}>
      {formatCompactDateTime(iso)}
    </time>
  );
}
