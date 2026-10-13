import Link from "next/link";

export default function NotFound() {
  return (
    <div className="space-y-4">
      <h1 className="text-2xl font-bold tracking-tight">Not found</h1>
      <p className="text-zinc-600 dark:text-zinc-400">
        There&apos;s no page at this address.
      </p>
      <Link href="/" className="font-medium underline underline-offset-4">
        Back to all posts
      </Link>
    </div>
  );
}
