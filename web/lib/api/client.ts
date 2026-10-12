import "server-only";

import createClient from "openapi-fetch";

import { apiErrorFrom } from "./errors";
import type { paths } from "./schema";

// Read per call, not at module load: next build imports this module to
// analyze pages, and must not need API_URL to do so.
export function apiClient() {
  const baseUrl = process.env.API_URL;
  if (!baseUrl) {
    throw new Error(
      "API_URL is not set; point it at the Go API, e.g. http://localhost:8080",
    );
  }
  return createClient<paths>({ baseUrl });
}

type Result<T> = { data?: T; error?: unknown; response: Response };

// Checks response.ok rather than error: openapi-fetch leaves error undefined
// for an empty error body.
export function unwrap<T>({ data, error, response }: Result<T>): T {
  if (!response.ok || data === undefined) {
    throw apiErrorFrom(response.status, error);
  }
  return data;
}
