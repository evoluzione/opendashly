FROM golang:1.25-alpine AS backend-build

WORKDIR /src/backend
COPY backend/go.mod backend/go.sum ./
RUN go mod download
COPY backend/ ./
RUN CGO_ENABLED=0 GOOS=linux go build -o /out/backend ./cmd/api

FROM node:20-alpine AS frontend-build

WORKDIR /src/frontend
COPY frontend/package.json frontend/package-lock.json ./
RUN npm ci
COPY frontend/ ./
ARG VITE_API_BASE=http://localhost:8080
ENV VITE_API_BASE=$VITE_API_BASE
RUN npm run build

FROM node:20-alpine

WORKDIR /app
COPY --from=backend-build /out/backend /app/backend
WORKDIR /app/frontend
COPY frontend/package.json frontend/package-lock.json ./
RUN npm ci --omit=dev
COPY --from=frontend-build /src/frontend/build /app/frontend/build
COPY docker/entrypoint.sh /app/entrypoint.sh
RUN chmod +x /app/entrypoint.sh

ENV API_LISTEN_ADDR=:8080
ENV NODE_ENV=production
ENV PORT=5173
ENV HOST=0.0.0.0

EXPOSE 8080 5173
ENTRYPOINT ["/app/entrypoint.sh"]
