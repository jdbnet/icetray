# Build static marketing site
FROM node:22-alpine AS builder
ARG VERSION=0.0.0
WORKDIR /app/website
COPY website/package.json website/package-lock.json* ./
RUN npm ci
COPY website/ .
ENV VITE_APP_VERSION=${VERSION}
RUN npm run build

FROM nginx:alpine
COPY website/nginx.conf /etc/nginx/conf.d/default.conf
COPY --from=builder /app/website/dist /usr/share/nginx/html
EXPOSE 80
