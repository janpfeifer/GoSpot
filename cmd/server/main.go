package main

import (
	"context"
	"log"

	"github.com/janpfeifer/GoSpot/internal/server"
)

func main() {
	if err := server.Run(context.Background(), ":8080"); err != nil {
		log.Fatal(err)
	}
}
