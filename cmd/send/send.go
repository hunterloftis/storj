package main

import (
	"errors"
	"fmt"
	"log"
	"os"

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
	args := os.Args[1:]

	filename := args[1]
	file, err := os.Open(filename)
	if err != nil {
		return fmt.Errorf("opening file %v: %w", filename, err)
	}
	defer file.Close()

	addr := args[0]
	client := relay.NewClient(addr, true)

	secret, wait, err := client.Send(filename, file)
	if err != nil {
		return fmt.Errorf("creating stream: %w", err)
	}
	fmt.Println(secret)

	return wait()
}
