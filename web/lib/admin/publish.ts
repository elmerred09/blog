// A post is a draft (published_at null), published (published_at in the
// past), or scheduled (in the future). The editor shows times in the
// browser's zone and sends UTC.

export type PublishMode = "draft" | "now" | "schedule";

// scheduleLocal is a datetime-local input's value ("2026-10-20T09:30"),
// meaningful only in schedule mode.
export type PublishState = {
  mode: PublishMode;
  scheduleLocal: string;
};

export type PublishedAt =
  { ok: true; value: string | null } | { ok: false; message: string };

export function initialPublishState(
  publishedAt: string | null,
  now: Date,
): PublishState {
  if (publishedAt === null) {
    return { mode: "draft", scheduleLocal: "" };
  }
  if (new Date(publishedAt) <= now) {
    return { mode: "now", scheduleLocal: "" };
  }
  return { mode: "schedule", scheduleLocal: toLocalInput(publishedAt) };
}

// publishedAtFor is the published_at to save. original is the post's current
// value: a post that's already live keeps its date, so re-saving it doesn't
// move it to the top of the list.
export function publishedAtFor(
  state: PublishState,
  original: string | null,
  now: Date,
): PublishedAt {
  switch (state.mode) {
    case "draft":
      return { ok: true, value: null };
    case "now":
      if (original !== null && new Date(original) <= now) {
        return { ok: true, value: original };
      }
      return { ok: true, value: now.toISOString() };
    case "schedule": {
      // A datetime-local value has no zone, so Date parses it as local time.
      const at = new Date(state.scheduleLocal);
      if (Number.isNaN(at.getTime())) {
        return { ok: false, message: "Pick a date and time to publish." };
      }
      if (at <= now) {
        return {
          ok: false,
          message: "Pick a time in the future, or choose Publish now.",
        };
      }
      return { ok: true, value: at.toISOString() };
    }
  }
}

function pad(n: number): string {
  return String(n).padStart(2, "0");
}

// toLocalInput formats an instant as a datetime-local value in the browser's
// time zone.
export function toLocalInput(iso: string): string {
  const d = new Date(iso);
  return (
    `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())}` +
    `T${pad(d.getHours())}:${pad(d.getMinutes())}`
  );
}
