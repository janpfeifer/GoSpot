package main

import (
	"context"
	"flag"
	"fmt"
	"log"

	"github.com/janpfeifer/GoSpot/internal/server"
)

var (
	flagAddr = flag.String("addr", "", "Address to listen on (default: auto-port on localhost)")
)

func main() {
	flag.Parse()

	started := make(chan *server.ServerState, 1)
	ctx := context.Background()

	go func() {
		state := <-started
		fmt.Printf("GoSpot server listening on http://%s\n", state.Address)
	}()

	if err := server.Run(ctx, *flagAddr, started); err != nil {
		log.Fatal(err)
	}
}
