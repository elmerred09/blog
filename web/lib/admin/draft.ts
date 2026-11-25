import type { PublishState } from "@/lib/admin/publish";

// The editor fields a save would store. Compared against the last load or
// save to decide whether Back would throw work away.
export type Draft = {
  title: string;
  slug: string;
  summary: string;
  body: string;
  tags: string[];
  publish: PublishState;
};

export function sameDraft(a: Draft, b: Draft): boolean {
  return (
    a.title === b.title &&
    a.slug === b.slug &&
    a.summary === b.summary &&
    a.body === b.body &&
    sameTags(a.tags, b.tags) &&
    samePublish(a.publish, b.publish)
  );
}

function sameTags(a: string[], b: string[]): boolean {
  if (a.length !== b.length) {
    return false;
  }
  for (let i = 0; i < a.length; i++) {
    if (a[i] !== b[i]) {
      return false;
    }
  }
  return true;
}

// A schedule time only matters while the post is set to schedule. Switching
// back to draft or publish-now leaves the datetime-local value behind.
function samePublish(a: PublishState, b: PublishState): boolean {
  return (
    a.mode === b.mode &&
    (a.mode !== "schedule" || a.scheduleLocal === b.scheduleLocal)
  );
}
