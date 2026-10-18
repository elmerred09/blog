import { getSessionCookie } from "better-auth/cookies";
import { NextResponse, type NextRequest } from "next/server";

// Sends visitors with no session cookie straight to sign-in. It only checks
// that a cookie is present, not that it's valid or allowed: requireAdmin does
// that in every admin page and server action.
export function proxy(request: NextRequest) {
  if (!getSessionCookie(request)) {
    return NextResponse.redirect(new URL("/sign-in", request.url));
  }
}

export const config = {
  matcher: ["/admin/:path*"],
};
