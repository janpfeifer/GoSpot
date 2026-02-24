package server

import (
	"context"
	"log"
	"net/http"
	"time"

	"github.com/janpfeifer/GoSpot/internal/lobby"
	"github.com/maxence-charriere/go-app/v10/pkg/app"
)

// Run starts the server and blocks until the context is canceled.
func Run(ctx context.Context, addr string) error {
	// Initialize global lobby state for server-side prerendering without panic
	lobby.InitState()

	// Initialize server state
	serverState := NewServerState()

	// Register go-app routes so the server knows how to prerender them
	app.Route("/", func() app.Composer { return &lobby.Home{} })
	app.RouteWithRegexp("^/table/.*", func() app.Composer { return &lobby.Table{} })

	// The web assets and the compiled webassembly
	// are served natively by the go-app framework
	h := &app.Handler{
		Name:        "GoSpot",
		Description: "A real-time matching game",
		Styles: []string{
			"/web/css/pico.min.css", // Load pico.css
			"/web/css/main.css",     // Custom styles if any
		},
	}

	mux := http.NewServeMux()

	// Register WebSocket endpoint
	mux.HandleFunc("/ws", serverState.HandleWS)

	// Serve the go-app UI
	// We want to serve /web for static files
	mux.Handle("/web/", http.StripPrefix("/web/", http.FileServer(http.Dir("web/"))))
	mux.Handle("/", h)

	srv := &http.Server{
		Addr:    addr,
		Handler: mux,
	}

	go func() {
		log.Printf("Server started on %s", addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("Server error: %v", err)
		}
	}()

	<-ctx.Done()

	// Graceful shutdown with 5 second timeout
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	log.Println("Shutting down server...")
	return srv.Shutdown(shutdownCtx)
}
