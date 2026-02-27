package server

import (
	"context"
	"net"
	"net/http"
	"time"

	"github.com/janpfeifer/GoSpot/internal/frontend"
	"github.com/janpfeifer/GoSpot/internal/game"
	"github.com/maxence-charriere/go-app/v10/pkg/app"
	"k8s.io/klog/v2"
)

// NetPipeAddr is the special address value that indicates to use net.Pipe() for testing.
const NetPipeAddr = "netpipe"

type pipeListener struct {
	conns  chan net.Conn
	closed chan struct{}
}

func (l *pipeListener) Accept() (net.Conn, error) {
	select {
	case c := <-l.conns:
		return c, nil
	case <-l.closed:
		return nil, net.ErrClosed
	}
}

func (l *pipeListener) Close() error {
	close(l.closed)
	return nil
}

func (l *pipeListener) Addr() net.Addr {
	return pipeAddr{}
}

type pipeAddr struct{}

func (pipeAddr) Network() string { return "pipe" }
func (pipeAddr) String() string  { return NetPipeAddr }

func (l *pipeListener) dial() (net.Conn, error) {
	select {
	case <-l.closed:
		return nil, net.ErrClosed
	default:
	}

	client, server := net.Pipe()
	select {
	case l.conns <- server:
		return client, nil
	case <-l.closed:
		return nil, net.ErrClosed
	}
}

// Run starts the server and blocks until the context is canceled.
// If addr is empty, it listens on an automatic port on the localhost interface.
// It sends the actual address it's listening on to the started channel if it's not nil.
func Run(ctx context.Context, addr string, started chan<- *ServerState) error {
	if addr == "" {
		addr = "127.0.0.1:0"
	}

	serverState := NewServerState()
	var ln net.Listener
	var err error
	if addr == NetPipeAddr {
		pl := &pipeListener{
			conns:  make(chan net.Conn),
			closed: make(chan struct{}),
		}
		serverState.LocalDial = pl.dial
		ln = pl
	} else {
		ln, err = net.Listen("tcp", addr)
	}
	if err != nil {
		return err
	}
	actualAddr := ln.Addr().String()

	// Initialize global lobby state for server-side prerendering without panic
	frontend.InitState()

	// Initialize server state
	serverState.Address = actualAddr
	if started != nil {
		started <- serverState
	}

	// Register go-app routes so the server knows how to prerender them
	app.Route("/", func() app.Composer { return &frontend.Home{} })
	app.RouteWithRegexp("^/table/.*", func() app.Composer { return &frontend.Table{} })

	// The web assets and the compiled webassembly
	// are served natively by the go-app framework
	defaultHandler := &app.Handler{
		Name:        "GoSpot",
		Description: "A real-time matching game",
		Styles: []string{
			"/web/css/pico.min.css", // Load pico.css
			"/web/css/main.css",     // Custom styles if any
		},
		Version: game.Version,
	}

	mux := http.NewServeMux()

	// Register /web/app.wasm explicitly to bypass the FileServer catch-all
	mux.HandleFunc("/web/app.wasm", func(w http.ResponseWriter, r *http.Request) {
		klog.Infof("Serving app.wasm")
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

	// Register test game endpoint
	mux.HandleFunc("/test/game", serverState.HandleTestGame)
	mux.HandleFunc("/test/game/", serverState.HandleTestGame)

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
