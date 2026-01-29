# Base image with pnpm
FROM node:20-slim AS base
ENV PNPM_HOME="/pnpm"
ENV PATH="$PNPM_HOME:$PATH"
RUN corepack enable

# Build stage
FROM base AS build
WORKDIR /app

# Copy workspace configuration and source files
COPY pnpm-workspace.yaml package.json ./
COPY .npmrc* ./
COPY packages/interfaces ./packages/interfaces
COPY apps/web ./apps/web

# Install all dependencies
RUN --mount=type=cache,id=pnpm,target=/pnpm/store pnpm install

# Build arguments for environment variables
ARG JAN_API_BASE_URL
ARG VITE_AUTH_URL
ARG VITE_AUTH_REALM
ARG VITE_AUTH_CLIENT_ID
ARG VITE_OAUTH_REDIRECT_URI
ARG VITE_GA_ID
ARG VITE_REPORT_ISSUE_URL
ARG POSTHOG_ENABLED
ARG POSTHOG_DEBUG
ARG POSTHOG_BATCH_SIZE
ARG POSTHOG_FLUSH_INTERVAL
ARG ANALYTICS_PII_LEVEL
ARG ANALYTICS_ENVIRONMENT
ARG POSTHOG_API_KEY

ENV JAN_API_BASE_URL=$JAN_API_BASE_URL
ENV VITE_AUTH_URL=$VITE_AUTH_URL
ENV VITE_AUTH_REALM=$VITE_AUTH_REALM
ENV VITE_AUTH_CLIENT_ID=$VITE_AUTH_CLIENT_ID
ENV VITE_OAUTH_REDIRECT_URI=$VITE_OAUTH_REDIRECT_URI
ENV VITE_GA_ID=$VITE_GA_ID
ENV VITE_REPORT_ISSUE_URL=$VITE_REPORT_ISSUE_URL
ENV POSTHOG_ENABLED=$POSTHOG_ENABLED
ENV POSTHOG_DEBUG=$POSTHOG_DEBUG
ENV POSTHOG_BATCH_SIZE=$POSTHOG_BATCH_SIZE
ENV POSTHOG_FLUSH_INTERVAL=$POSTHOG_FLUSH_INTERVAL
ENV ANALYTICS_PII_LEVEL=$ANALYTICS_PII_LEVEL
ENV ANALYTICS_ENVIRONMENT=$ANALYTICS_ENVIRONMENT
ARG POSTHOG_API_KEY=$POSTHOG_API_KEY

# Build the interfaces package first
RUN pnpm build:interfaces

# Build the web app
RUN pnpm build:web

# Production stage
FROM nginx:alpine AS runner

# Copy nginx configuration
COPY apps/web/nginx.conf /etc/nginx/nginx.conf

# Copy built files
COPY --from=build /app/apps/web/dist /usr/share/nginx/html

# Health check
HEALTHCHECK --interval=30s --timeout=10s --retries=3 --start-period=40s \
  CMD wget -q --spider http://localhost:3001/health || exit 1

EXPOSE 3001

CMD ["nginx", "-g", "daemon off;"]
