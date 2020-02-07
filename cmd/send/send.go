package main

import (
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/hunterloftis/storj/relay"
)

func main() {
	if err := send(); err != nil {
		log.Fatalf("error: %v", err)
	}
}

func send() error {
	if len(os.Args) < 3 {
		return errors.New("insufficient arguments")
	}

	addr := os.Args[1]
	filename := os.Args[2]

	_, name := filepath.Split(filename)
	file, err := os.Open(filename)
	if err != nil {
		return fmt.Errorf("opening file %v: %w", filename, err)
	}
	defer file.Close()

	client := relay.NewClient(addr)
	secret, send, err := client.Offer(name, file)
	if err != nil {
		return fmt.Errorf("creating stream: %w", err)
	}

	fmt.Println(secret)
	return send()
}
