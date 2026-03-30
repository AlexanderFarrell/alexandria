# Stage 1: Build Vue 3 frontend
FROM node:20-alpine AS frontend
WORKDIR /app/client
COPY client/package*.json ./
RUN npm ci
COPY client .
RUN npm run build

# Stage 2: Build Go server
FROM golang:1.25-alpine AS backend
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
# CGO disabled — using pure-Go SQLite driver (glebarez/sqlite)
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o server ./cmd/server

# Stage 3: Minimal runtime image
FROM alpine:latest
RUN apk --no-cache add ca-certificates tzdata
WORKDIR /app
COPY --from=backend /app/server .
COPY --from=frontend /app/client/dist ./client/dist

EXPOSE 8080
ENV PORT=8080 \
    DATA_DIR=/data \
    DB_PATH=/data/alexandria.db

VOLUME ["/data"]

CMD ["./server"]
