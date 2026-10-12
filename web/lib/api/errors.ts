import type { ErrorCode } from "./types";

// unexpected_response: a non-2xx whose body isn't the API's error JSON, such
// as a reverse proxy's 502 page.
export type ApiErrorCode = ErrorCode | "unexpected_response";

export class ApiError extends Error {
  readonly status: number;
  readonly code: ApiErrorCode;

  constructor(status: number, code: ApiErrorCode, message: string) {
    super(message);
    this.name = "ApiError";
    this.status = status;
    this.code = code;
  }
}

export function apiErrorFrom(status: number, body: unknown): ApiError {
  if (isErrorBody(body)) {
    return new ApiError(status, body.error.code, body.error.message);
  }
  return new ApiError(
    status,
    "unexpected_response",
    `unexpected ${status} response from the API`,
  );
}

function isErrorBody(
  body: unknown,
): body is { error: { code: ErrorCode; message: string } } {
  if (typeof body !== "object" || body === null || !("error" in body)) {
    return false;
  }
  const { error } = body;
  return (
    typeof error === "object" &&
    error !== null &&
    "code" in error &&
    typeof error.code === "string" &&
    "message" in error &&
    typeof error.message === "string"
  );
}
