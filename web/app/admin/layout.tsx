import type { Metadata } from "next";

export const metadata: Metadata = {
  title: { default: "Admin", template: "%s | Admin" },
  robots: { index: false, follow: false },
};

// The admin links and sign-out live in the site header (AdminNav). Pages
// call requireAdmin themselves: a layout check doesn't guard a page's data or
// its server actions.
export default function AdminLayout({ children }: LayoutProps<"/admin">) {
  return children;
}
