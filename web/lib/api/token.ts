import "server-only";

import { importPKCS8, SignJWT, type CryptoKey } from "jose";

// internal/auth rejects a token whose exp is more than five minutes out. Each
// token covers one request, so it only needs to outlive clock skew.
export const TOKEN_TTL_SECONDS = 120;

let imported: { pem: string; key: Promise<CryptoKey> } | undefined;

// Read per call, like API_URL, and re-imported only when the value changes.
function privateKey(): Promise<CryptoKey> {
  const raw = process.env.ADMIN_JWT_PRIVATE_KEY;
  if (!raw) {
    throw new Error(
      "ADMIN_JWT_PRIVATE_KEY is not set; run make admin-keys to create it",
    );
  }
  // .env files written by make admin-keys escape the newlines. Next's loader
  // expands them; a shell export doesn't.
  const pem = raw.replaceAll("\\n", "\n");
  if (imported?.pem !== pem) {
    imported = { pem, key: importPKCS8(pem, "EdDSA") };
  }
  return imported.key;
}

// signApiToken signs the bearer token for one admin API request. The subject
// matches the API's ADMIN_SUBJECTS entries.
export async function signApiToken(
  githubId: string,
  now: Date = new Date(),
): Promise<string> {
  const iat = Math.floor(now.getTime() / 1000);
  return new SignJWT()
    .setProtectedHeader({ alg: "EdDSA", typ: "JWT" })
    .setIssuer("blog-web")
    .setAudience("blog-api")
    .setSubject(`github:${githubId}`)
    .setIssuedAt(iat)
    .setExpirationTime(iat + TOKEN_TTL_SECONDS)
    .sign(await privateKey());
}
