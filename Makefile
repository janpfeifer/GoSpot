.PHONY: build run clean build-wasm build-server

# Build everything
build: build-wasm build-server

# Build the WebAssembly frontend and place it in the web/ directory
build-wasm:
	@echo "Building Wasm frontend..."
	GOOS=js GOARCH=wasm go build -o web/app.wasm ./cmd/wasm

# Build the Go backend server
build-server:
	@echo "Building Go backend..."
	go build -o bin/server ./cmd/server

# Run the server (building it first to ensure it's up to date)
run: build
	@echo "Starting GoSpot server on http://localhost:8080..."
	./bin/server

# Clean up build artifacts
clean:
	@echo "Cleaning up..."
	rm -f web/app.wasm
	rm -rf bin/
