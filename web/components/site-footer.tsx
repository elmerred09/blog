import { SocialLinks } from "@/components/social-links";
import { siteName } from "@/lib/site";

export function SiteFooter() {
  const year = process.env.BUILD_YEAR;
  return (
    <footer className="mx-auto w-full max-w-[calc(960px+3rem)] px-6 pb-10">
      <div className="flex items-center justify-between gap-4 border-t border-line pt-8">
        <p className="text-sm text-muted">
          © {year ? `${year} ` : ""}
          {siteName}
        </p>
        <SocialLinks />
      </div>
    </footer>
  );
}
