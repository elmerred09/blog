# Next standalone image. Build from web/:
#   docker build -f ../deploy/docker/web.Dockerfile --build-arg SITE_URL=https://example.com .

FROM node:24-slim AS build
WORKDIR /app
COPY package.json package-lock.json ./
RUN npm ci
COPY . .
# robots.txt and the prerendered metadata read these while next build runs.
ARG SITE_URL
ARG SITE_INDEXING=false
ENV SITE_URL=$SITE_URL
ENV SITE_INDEXING=$SITE_INDEXING
ENV NEXT_TELEMETRY_DISABLED=1
RUN npm run build

FROM node:24-slim AS run
WORKDIR /app
ENV NODE_ENV=production
ENV PORT=3000
ENV HOSTNAME=0.0.0.0
RUN useradd --system --uid 1001 --create-home --shell /usr/sbin/nologin nextjs
COPY --from=build --chown=nextjs:nextjs /app/.next/standalone ./
COPY --from=build --chown=nextjs:nextjs /app/.next/static ./.next/static
COPY --from=build --chown=nextjs:nextjs /app/public ./public
# og.tsx reads the fonts from process.cwd()/assets/fonts. Standalone output
# does not trace them.
COPY --from=build --chown=nextjs:nextjs /app/assets/fonts ./assets/fonts
USER nextjs
EXPOSE 3000
CMD ["node", "server.js"]
