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

# Remove default nginx configuration and copy custom config
RUN rm /etc/nginx/conf.d/default.conf
COPY nginx.conf /etc/nginx/conf.d/keystone.conf

# Copy built React app from builder stage
COPY --from=builder /app/build /usr/share/nginx/html

EXPOSE 3000

CMD ["nginx", "-g", "daemon off;"]
