package integration

import (
	"fmt"
	"io/ioutil"
	"math/rand"
	"strings"
	"testing"
	"time"

	"github.com/hunterloftis/storj/relay"
)

func TestIntegrationSimple(t *testing.T) {
	rng := rand.New(rand.NewSource(time.Now().UnixNano()))
	secrets := relay.NewSecrets(rng)
	handler := relay.NewHandler(secrets, ioutil.Discard)
	addr := "localhost:3000"

	server, err := relay.NewServer(addr, handler)
	if err != nil {
		t.Errorf("creating server: %w", err)
	}

	go func() {
		if err := server.ListenAndServeTLS("", ""); err != nil {
			t.Errorf("starting server: %w", err)
		}
	}()

	filename := "file.txt"
	contents := "file contents"
	file := ioutil.NopCloser(strings.NewReader(contents))

	sender := relay.NewClient(addr, true)
	secret, _, err := sender.Send(filename, file)
	if err != nil {
		t.Errorf("opening send stream: %w", err)
	}

	receiver := relay.NewClient(addr, true)
	suggestedName, stream, err := receiver.Receive(secret)
	if err != nil {
		t.Errorf("receiving: %w", err)
	}

	received, err := ioutil.ReadAll(stream)
	if err != nil {
		t.Error("reading stream:", err)
	}

	t.Run("suggests a filename", func(t *testing.T) {
		got := suggestedName
		want := filename

		if got != want {
			t.Errorf("got %q, want %q", got, want)
		}
	})

	t.Run("receives the file", func(t *testing.T) {
		got := fmt.Sprintf("%s", received)
		want := contents

		if got != want {
			t.Errorf("got %q, want %q", got, want)
		}
	})
}
