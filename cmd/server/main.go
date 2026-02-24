package main

import (
	"context"
	"flag"
	"fmt"
	"log"

	"github.com/janpfeifer/GoSpot/internal/server"
)

func main() {
	var addr string
	flag.StringVar(&addr, "addr", "", "Address to listen on (default: auto-port on localhost)")
	flag.Parse()

	started := make(chan string, 1)
	ctx := context.Background()

	go func() {
		actualAddr := <-started
		fmt.Printf("GoSpot server listening on http://%s\n", actualAddr)
	}()

	if err := server.Run(ctx, addr, started); err != nil {
		log.Fatal(err)
	}
}
