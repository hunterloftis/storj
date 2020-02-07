package main

import (
	"errors"
	"log"
	"math/rand"
	"net/http"
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
	secrets := relay.NewSecrets(rng)
	handler := relay.NewHandler(secrets, os.Stdout)

	return http.ListenAndServe(addr, handler)
}
