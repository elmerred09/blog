import { getAuth } from "@/lib/auth";

// Better Auth's endpoints: the GitHub redirect and callback, session reads,
// and sign-out. Without admin config they don't exist, like /admin.
function handle(request: Request) {
  const configured = getAuth();
  if (!configured) {
    return new Response("Not Found", { status: 404 });
  }
  return configured.auth.handler(request);
}

export { handle as GET, handle as POST };
