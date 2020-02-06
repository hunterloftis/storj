package main

import (
	"errors"
	"fmt"
	"log"
	"math/rand"
	"os"
	"time"

	"github.com/hunterloftis/storj/relay"
)

func main() {
	if err := start(); err != nil {
		log.Fatalf("error: %v", err)
	}
}

func start() error {
	if len(os.Args) < 2 {
		return errors.New("insufficient arguments")
	}

	addr := os.Args[1]
	rng := rand.New(rand.NewSource(time.Now().UnixNano()))
	server, err := relay.NewServer(addr, rng)
	if err != nil {
		return fmt.Errorf("creating server: %w", err)
	}

	return server.ListenAndServeTLS("", "")
}
