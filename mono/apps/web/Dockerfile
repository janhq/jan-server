# Jan Server Mono - Web Frontend Dockerfile
#
# This Dockerfile serves the pre-built React application.
# For development, use: npm run dev
#
# To rebuild the web app:
#   cd apps/web && npm install && npm run build

# Production stage - serve with nginx
FROM nginx:alpine

# Copy nginx configuration
COPY apps/web/nginx.conf /etc/nginx/nginx.conf

# Copy pre-built files
COPY apps/web/dist /usr/share/nginx/html

# Health check
HEALTHCHECK --interval=30s --timeout=10s --retries=3 --start-period=10s \
  CMD wget -q --spider http://localhost:80/health || exit 1

EXPOSE 80

CMD ["nginx", "-g", "daemon off;"]
