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

	rng := rand.New(rand.NewSource(time.Now().UnixNano()))
	secrets := relay.NewSecrets(rng)
	handler := relay.NewHandler(secrets, os.Stdout)
	addr := os.Args[1]
	server, err := relay.NewServer(addr, handler)
	if err != nil {
		return fmt.Errorf("creating server: %w", err)
	}

	return server.ListenAndServeTLS("", "")
}
