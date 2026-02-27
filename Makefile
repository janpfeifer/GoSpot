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

docker-build:
	@echo -n "Building Docker image: -> " ; grep 'var Version =' internal/game/game.go
	docker build . --tag us-central1-docker.pkg.dev/gospot-488708/gospot-repo/gospot:latest
	@echo "Pushing Docker image..."
	docker push us-central1-docker.pkg.dev/gospot-488708/gospot-repo/gospot:latest

# Run the server (building it first to ensure it's up to date)
run: build
	@echo "Starting GoSpot server on http://localhost:8080..."
	./bin/server -addr=:8080 -dev

gcloud-run:
	@echo "(Re)-starting to Google Cloud Run..."
	gcloud run deploy gospot \
		--image=us-central1-docker.pkg.dev/gospot-488708/gospot-repo/gospot:latest \
		--region=us-central1 \
		--allow-unauthenticated \
		--max-instances=1 \
		--port=8080

# Clean up build artifacts
clean:
	@echo "Cleaning up..."
	rm -f web/app.wasm
	rm -rf bin/
