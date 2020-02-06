package main

import (
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"

	"github.com/hunterloftis/storj/relay"
)

func main() {
	if err := receive(); err != nil {
		log.Fatalf("error: %v", err)
	}
}

func receive() error {
	if len(os.Args) < 4 {
		return errors.New("insufficient arguments")
	}
	args := os.Args[1:]

	addr := args[0]
	client := relay.NewClient(addr, true)

	secret := args[1]
	suggestedName, body, err := client.Receive(secret)
	if err != nil {
		return fmt.Errorf("opening receive stream: %w", err)
	}

	// TODO: mkdirp if dir filename path doesn't exist
	dir := args[2]
	_, name := filepath.Split(suggestedName) // TODO: is this sufficiently secure?
	filename := filepath.Join(dir, name)
	file, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("writing to file %v: %w", filename, err)
	}
	defer file.Close()

	if _, err := io.Copy(file, body); err != nil {
		return fmt.Errorf("streaming file: %w", err)
	}
	return nil
}
