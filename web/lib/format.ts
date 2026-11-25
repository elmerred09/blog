// A fixed locale and time zone, so the server-rendered text matches what the
// browser would produce during hydration.
const dateFormat = new Intl.DateTimeFormat("en-US", {
  dateStyle: "medium",
  timeZone: "UTC",
});

const dateTimeFormat = new Intl.DateTimeFormat("en-US", {
  dateStyle: "medium",
  timeStyle: "short",
  timeZone: "UTC",
});

export function formatDate(iso: string): string {
  return dateFormat.format(new Date(iso));
}

export function formatReadingTime(minutes: number): string {
  return `${minutes} min read`;
}

export function formatDateTime(iso: string): string {
  return `${dateTimeFormat.format(new Date(iso))} UTC`;
}

// The two below default to the runtime's own time zone, so on the server they
// give UTC. Call them only from client-only rendering (see LocalTime).
type ZoneOptions = { timeZone?: string };

// "Oct 18, 4:05 PM" within the current year, "Oct 18, 2025" outside it.
export function formatCompactDateTime(
  iso: string,
  { now = new Date(), timeZone }: ZoneOptions & { now?: Date } = {},
): string {
  const date = new Date(iso);
  const year = new Intl.DateTimeFormat("en-US", { year: "numeric", timeZone });
  if (year.format(date) !== year.format(now)) {
    return new Intl.DateTimeFormat("en-US", {
      dateStyle: "medium",
      timeZone,
    }).format(date);
  }
  return new Intl.DateTimeFormat("en-US", {
    month: "short",
    day: "numeric",
    hour: "numeric",
    minute: "2-digit",
    timeZone,
  }).format(date);
}

// "Oct 18, 2026, 12:05 PM EDT"
export function formatZonedDateTime(
  iso: string,
  { timeZone }: ZoneOptions = {},
): string {
  return new Intl.DateTimeFormat("en-US", {
    month: "short",
    day: "numeric",
    year: "numeric",
    hour: "numeric",
    minute: "2-digit",
    timeZoneName: "short",
    timeZone,
  }).format(new Date(iso));
}
