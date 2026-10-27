import type { Metadata } from "next";
import { Geist_Mono, Inter } from "next/font/google";
import Link from "next/link";

import { Menu } from "@/components/menu";
import { siteName, siteUrl } from "@/lib/site";

import "./globals.css";

const inter = Inter({
  variable: "--font-inter",
  subsets: ["latin"],
});

const geistMono = Geist_Mono({
  variable: "--font-geist-mono",
  subsets: ["latin"],
});

const description = "A technical blog about software development.";

export const metadata: Metadata = {
  metadataBase: new URL(siteUrl()),
  title: { default: siteName, template: `%s | ${siteName}` },
  description,
  openGraph: {
    siteName,
    type: "website",
    locale: "en_US",
    description,
  },
  twitter: { card: "summary_large_image" },
};

export default function RootLayout({ children }: LayoutProps<"/">) {
  return (
    <html
      lang="en"
      className={`${inter.variable} ${geistMono.variable} h-full antialiased`}
    >
      <body className="flex min-h-full flex-col font-sans">
        <header className="mx-auto flex w-full max-w-6xl items-center justify-between px-6 pt-8">
          <Link href="/" className="text-xl font-semibold tracking-tight">
            {siteName}
          </Link>
          <nav>
            <Menu />
          </nav>
        </header>
        <main className="mx-auto w-full max-w-6xl flex-1 px-6 pt-24 pb-12">
          {children}
        </main>
      </body>
    </html>
  );
}
