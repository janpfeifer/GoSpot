package server

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"
)

func TestServerRun(t *testing.T) {
	// Use a background context that we can cancel
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	addr := "127.0.0.1:8081"

	// Start the server in a goroutine
	errCh := make(chan error, 1)
	go func() {
		errCh <- Run(ctx, addr)
	}()

	// Wait a moment for the server to start
	time.Sleep(500 * time.Millisecond)

	// Make an HTTP request to the login page (root route maps to login for unauthenticated)
	resp, err := http.Get("http://" + addr + "/")
	if err != nil {
		t.Fatalf("Failed to connect to server: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status OK, got %v", resp.Status)
	}

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("Failed to read body: %v", err)
	}

	body := string(bodyBytes)

	// The go-app framework generates standard HTML. Let's check if our title or something is in there
	// The word GoSpot should definitely be there.
	if !strings.Contains(body, "GoSpot") {
		t.Errorf("Expected body to contain 'GoSpot', got body: %s", body)
	}

	// Cancel the context to stop the server
	cancel()

	// Wait for the server to shutdown cleanly
	select {
	case err := <-errCh:
		if err != nil {
			t.Errorf("Server shut down with error: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Errorf("Server took too long to shut down")
	}
}
