FROM node:18-alpine AS builder

WORKDIR /app

# Accept build-time environment variables
ARG NEXT_PUBLIC_BACKEND_URL
ARG NEXT_PUBLIC_WEBSOCKET_URL

# Set them as environment variables for the build process
ENV NEXT_PUBLIC_BACKEND_URL=${NEXT_PUBLIC_BACKEND_URL}
ENV NEXT_PUBLIC_WEBSOCKET_URL=${NEXT_PUBLIC_WEBSOCKET_URL}

COPY package*.json ./
COPY . .

RUN npm ci
RUN npm run build

FROM node:18-alpine AS production

WORKDIR /app

ENV NODE_ENV=production
ENV NEXT_TELEMETRY_DISABLED=1

RUN addgroup -g 1001 -S nodejs
RUN adduser -S nextjs -u 1001

COPY --from=builder /app/public ./public
COPY --from=builder /app/.next/standalone ./
COPY --from=builder /app/.next/static ./.next/static

USER nextjs

EXPOSE 3000

ENV PORT=3000
ENV HOSTNAME="0.0.0.0"

CMD ["node", "server.js"]
