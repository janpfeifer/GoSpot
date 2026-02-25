package server

import (
	"context"
	"net"
	"net/http"
	"time"

	"github.com/janpfeifer/GoSpot/internal/lobby"
	"github.com/maxence-charriere/go-app/v10/pkg/app"
	"k8s.io/klog/v2"
)

// Run starts the server and blocks until the context is canceled.
// If addr is empty, it listens on an automatic port on the localhost interface.
// It sends the actual address it's listening on to the started channel if it's not nil.
func Run(ctx context.Context, addr string, started chan<- string) error {
	if addr == "" {
		addr = "127.0.0.1:0"
	}

	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}
	actualAddr := ln.Addr().String()
	if started != nil {
		started <- actualAddr
	}

	// Initialize global lobby state for server-side prerendering without panic
	lobby.InitState()

	// Initialize server state
	serverState := NewServerState()

	// Register go-app routes so the server knows how to prerender them
	app.Route("/", func() app.Composer { return &lobby.Home{} })
	app.RouteWithRegexp("^/table/.*", func() app.Composer { return &lobby.Table{} })

	// The web assets and the compiled webassembly
	// are served natively by the go-app framework
	defaultHandler := &app.Handler{
		Name:        "GoSpot",
		Description: "A real-time matching game",
		Styles: []string{
			"/web/css/pico.min.css", // Load pico.css
			"/web/css/main.css",     // Custom styles if any
		},
	}

	mux := http.NewServeMux()

	// Register app.wasm explicitly since it's in /web/
	mux.HandleFunc("/app.wasm", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "web/app.wasm")
	})

	// Register logout handler
	mux.HandleFunc("/logout", func(w http.ResponseWriter, r *http.Request) {
		http.SetCookie(w, &http.Cookie{
			Name:   "gospot_player",
			Value:  "",
			Path:   "/",
			MaxAge: -1,
		})
		http.Redirect(w, r, "/", http.StatusSeeOther)
	})

	// Register WebSocket endpoint
	mux.HandleFunc("/ws", serverState.HandleWS)

	// Serve the go-app UI
	// We want to serve /web for static files
	mux.Handle("/web/", http.StripPrefix("/web/", http.FileServer(http.Dir("web/"))))
	mux.Handle("/", defaultHandler)

	srv := &http.Server{
		Handler: mux,
	}

	go func() {
		klog.Infof("Server started on %s", actualAddr)
		if err := srv.Serve(ln); err != nil && err != http.ErrServerClosed {
			klog.Infof("Server error: %v", err)
		}
	}()

	<-ctx.Done()

	// Graceful shutdown with 5 second timeout
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	klog.Infof("Shutting down server...")
	return srv.Shutdown(shutdownCtx)
}
