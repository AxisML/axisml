# Build the SPA with pnpm, then serve the static bundle with nginx.
# Build context is this component dir (the generated API client is committed
# under src/api/generated, so the build does not need the OpenAPI spec).

# --- build stage ---
FROM node:22-alpine AS build
WORKDIR /app
RUN corepack enable
# Install deps first for layer caching (pnpm version is pinned via packageManager).
COPY package.json pnpm-lock.yaml ./
RUN pnpm install --frozen-lockfile
COPY . .
RUN pnpm run build

# --- runtime stage ---
FROM nginx:1.27-alpine
COPY nginx.conf /etc/nginx/conf.d/default.conf
COPY --from=build /app/dist /usr/share/nginx/html
EXPOSE 80
CMD ["nginx", "-g", "daemon off;"]
