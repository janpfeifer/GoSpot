package main

import (
	"flag"
	"os"

	"github.com/janpfeifer/GoSpot/internal/frontend"
	"github.com/maxence-charriere/go-app/v10/pkg/app"
	"k8s.io/klog/v2"
)

func main() {
	// Initialize klog for WASM, forcing logs to stderr (console)
	fs := flag.NewFlagSet("klog", flag.ContinueOnError)
	klog.InitFlags(fs)
	fs.Set("logtostderr", "true")
	klog.SetOutput(os.Stderr)
	klog.Infof("WASM started!")

	// Root route handles both Login and Home page logic
	app.Route("/", func() app.Composer { return &frontend.Home{} })

	// Table route for a specific game room
	app.RouteWithRegexp("^/table/.*", func() app.Composer { return &frontend.Table{} })

	// Game route for a specific game room
	app.RouteWithRegexp("^/game/.*", func() app.Composer { return &frontend.Game{} })

	// Initialize the global app state manager
	frontend.InitState()

	// When building for WEB (GOOS=js GOARCH=wasm), app.Run() executes the frontend logic
	app.RunWhenOnBrowser()

	// In server mode, app.RunWhenOnBrowser doesn't do anything.
	// But our server is in cmd/server/, so we don't even reach here natively.
}
