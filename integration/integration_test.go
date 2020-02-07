package integration

import (
	"io/ioutil"
	"math/rand"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/hunterloftis/storj/relay"
)

func TestIntegrationSimple(t *testing.T) {
	addr := "localhost:3000"
	filename := "file.txt"
	contents := "file contents"

	go func() {
		rng := rand.New(rand.NewSource(time.Now().UnixNano()))
		secrets := relay.NewSecrets(rng)
		handler := relay.NewHandler(secrets, ioutil.Discard)

		if err := http.ListenAndServe(addr, handler); err != nil {
			t.Errorf("starting server: %w", err)
		}
	}()

	time.Sleep(100 * time.Millisecond)
	offering := make(chan struct{})
	var sharedSecret string

	go func() {
		file := ioutil.NopCloser(strings.NewReader(contents))

		sender := relay.NewClient(addr)
		secret, send, err := sender.Offer(filename, file)
		if err != nil {
			t.Errorf("opening send stream: %w", err)
		}

		sharedSecret = secret
		close(offering)
		if err := send(); err != nil {
			t.Errorf("send err: %w", err)
		}
	}()

	<-offering

	receiver := relay.NewClient(addr)
	suggestedName, stream, err := receiver.Receive(sharedSecret)
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
		got := string(received)
		want := contents

		if got != want {
			t.Errorf("got %q, want %q", got, want)
		}
	})
}
