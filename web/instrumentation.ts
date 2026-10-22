import { registerOTel } from "@vercel/otel";

// Next calls register once, before it serves. Without a collector configured,
// leave the SDK off so `next build`, tests, and CI export nothing.
export function register() {
  if (!process.env.OTEL_EXPORTER_OTLP_ENDPOINT) {
    return;
  }

  // A traceparent header is added for API_URL. @vercel/otel also adds one for
  // any http://localhost URL when the app is not on Vercel, which covers the
  // local API. It does not add one for GitHub or other third parties.
  const apiURL = process.env.API_URL;
  registerOTel({
    serviceName: "blog-web",
    instrumentationConfig: {
      fetch: {
        propagateContextUrls: apiURL ? [apiURL] : [],
      },
    },
  });
}
