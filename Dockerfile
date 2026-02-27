# Build the WebAssembly frontend and the Go server
FROM golang:1.26-alpine AS builder

WORKDIR /app
COPY . .

# Build the WASM file as defined in your Makefile
RUN GOOS=js GOARCH=wasm go build -o web/app.wasm ./cmd/wasm

# Build the server
RUN go build -o bin/server ./cmd/server

# Create the minimal runtime image
FROM alpine:latest
WORKDIR /app

# Copy the compiled binaries and web assets
COPY --from=builder /app/bin/server /app/server
COPY --from=builder /app/web /app/web

# Cloud Run sets the PORT environment variable (default 8080)
# We will run the server listening on 0.0.0.0 and the assigned port
CMD ["sh", "-c", "./server -addr 0.0.0.0:${PORT:-8080}"]