FROM node:22-alpine AS builder

WORKDIR /src

RUN corepack enable
RUN corepack prepare pnpm@10.10.0 --activate

ARG VITE_API_TIMEOUT_MS=120000
ARG VITE_UPLOAD_TIMEOUT_MS=600000
ENV VITE_API_TIMEOUT_MS=$VITE_API_TIMEOUT_MS \
    VITE_UPLOAD_TIMEOUT_MS=$VITE_UPLOAD_TIMEOUT_MS

COPY package.json pnpm-lock.yaml ./
RUN pnpm install --frozen-lockfile

COPY . .
RUN pnpm build

FROM nginx:1.27-alpine

ENV API_PROXY_READ_TIMEOUT=600s \
    API_PROXY_SEND_TIMEOUT=600s

COPY nginx.conf.template /etc/nginx/templates/default.conf.template
COPY --from=builder /src/dist /usr/share/nginx/html

EXPOSE 80
