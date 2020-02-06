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

	addr := os.Args[1]
	client := relay.NewClient(addr, true)
	secret := os.Args[2]

	suggestedName, stream, err := client.Receive(secret)
	if err != nil {
		return fmt.Errorf("opening receive stream: %w", err)
	}

	dir := os.Args[3]
	_, name := filepath.Split(suggestedName)
	filename := filepath.Join(dir, name)
	file, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("writing to file %v: %w", filename, err)
	}
	defer file.Close()

	if _, err := io.Copy(file, stream); err != nil {
		return fmt.Errorf("streaming file: %w", err)
	}
	return nil
}
