// The brand mark: a white "1" on a blue square. app/icon.svg is the same
// drawing with the colors written out; change both together. The "1" sits
// left of the box's center so its ink is centered, and its stem edges are
// whole units so they land on pixels at the nav's 32px.
export function LogoMark({ className }: { className?: string }) {
  return (
    <svg
      viewBox="0 0 32 32"
      aria-hidden="true"
      focusable="false"
      className={className}
    >
      <rect width="32" height="32" rx="5" className="fill-brand" />
      <path d="M15 8H19V24H15V12.5L11.5 14.8L10 11.6Z" className="fill-white" />
    </svg>
  );
}
