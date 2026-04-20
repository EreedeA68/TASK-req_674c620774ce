# =============================================================================
# Stage 1: Build React application
# =============================================================================
FROM node:18-alpine AS builder

WORKDIR /app

# Copy dependency manifests first for better layer caching
COPY package.json ./

# Install dependencies
RUN npm install --no-audit

# Copy the rest of the source code
COPY . .

# Build the production bundle
RUN npm run build

# =============================================================================
# Stage 2: Test runner (used by run_tests.sh via docker build --target test)
# =============================================================================
FROM builder AS test
CMD ["npx", "jest", "--coverage", "--coverageReporters=text", "--passWithNoTests"]

# =============================================================================
# Stage 3: Serve with Nginx
# =============================================================================
FROM nginx:alpine

# Remove default nginx configuration
RUN rm /etc/nginx/conf.d/default.conf

# Write custom nginx configuration inline for SPA routing
RUN cat > /etc/nginx/conf.d/keystone.conf << 'EOF'
server {
    listen 3000;
    server_name _;

    root /usr/share/nginx/html;
    index index.html;

    # Gzip compression
    gzip on;
    gzip_vary on;
    gzip_min_length 1024;
    gzip_proxied any;
    gzip_comp_level 6;
    gzip_types
        text/plain
        text/css
        text/xml
        text/javascript
        application/javascript
        application/x-javascript
        application/xml
        application/json
        application/rss+xml
        application/atom+xml
        image/svg+xml;

    # Security headers
    add_header X-Frame-Options "SAMEORIGIN" always;
    add_header X-Content-Type-Options "nosniff" always;
    add_header X-XSS-Protection "1; mode=block" always;
    add_header Referrer-Policy "strict-origin-when-cross-origin" always;
    add_header Content-Security-Policy "default-src 'self'; script-src 'self' 'unsafe-inline'; style-src 'self' 'unsafe-inline'; img-src 'self' data: blob:; font-src 'self'; connect-src 'self';" always;

    # Proxy API requests to backend
    location /api/ {
        proxy_pass http://keystone-backend:8080/api/;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
    }

    # Cache static assets
    location ~* \.(js|css|png|jpg|jpeg|gif|ico|svg|woff|woff2|ttf|eot)$ {
        expires 1y;
        add_header Cache-Control "public, immutable";
        try_files $uri =404;
    }

    # SPA catch-all: serve index.html for all unmatched routes
    location / {
        try_files $uri $uri/ /index.html;
    }

    # Healthcheck endpoint
    location /healthz {
        access_log off;
        return 200 "ok\n";
        add_header Content-Type text/plain;
    }
}
EOF

# Copy built React app from builder stage
COPY --from=builder /app/build /usr/share/nginx/html

EXPOSE 3000

CMD ["nginx", "-g", "daemon off;"]
