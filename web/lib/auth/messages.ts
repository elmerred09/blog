// signInErrorMessage turns the error code on /sign-in?error= into something
// to show. Codes come from the allowlist, the sign-in action, or Better
// Auth's OAuth callback (which also passes GitHub's own, like access_denied).
export function signInErrorMessage(code: string): string {
  switch (code) {
    case "not_allowed":
      return "That GitHub account isn't allowed to sign in here.";
    case "not_configured":
      return "Admin sign-in isn't configured on this server.";
    case "access_denied":
      return "GitHub sign-in was cancelled.";
    default:
      return "Sign-in failed. Try again.";
  }
}
