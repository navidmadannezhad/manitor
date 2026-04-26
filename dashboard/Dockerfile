# syntax=docker/dockerfile:1
FROM hub.hamdocker.ir/node:22-alpine AS build
WORKDIR /app
RUN corepack enable && corepack prepare pnpm@9 --activate
COPY package.json pnpm-lock.yaml ./
RUN pnpm install --frozen-lockfile
COPY . .
# Browser must reach the API; from the host, map server port 5000 (see docker-compose).
ARG VITE_SERVER_BASE_URL=http://localhost:5000
ENV VITE_SERVER_BASE_URL=$VITE_SERVER_BASE_URL
RUN pnpm run build

FROM hub.hamdocker.ir/nginx:1.27-alpine
COPY --from=build /app/dist /usr/share/nginx/html
COPY nginx.conf /etc/nginx/conf.d/default.conf
EXPOSE 80
HEALTHCHECK --interval=30s --timeout=3s --retries=3 \
  CMD wget -qO- http://127.0.0.1/ > /dev/null || exit 1
