# Stage 1: Build
FROM golang:1.22-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
# Static binary (mongo driver is pure Go → CGO off).
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /server ./cmd/server

# Stage 2: Run
# distroless/static bundles CA certs (REQUIRED for the MongoDB Atlas TLS dial)
# and tzdata, and runs as a non-root user — no package manager / shell needed.
FROM gcr.io/distroless/static-debian12:nonroot

COPY --from=builder /server /server

# App Service is pointed here via WEBSITES_PORT; the app reads $PORT (default 8080).
EXPOSE 8080
ENV PORT=8080
USER nonroot:nonroot

ENTRYPOINT ["/server"]
