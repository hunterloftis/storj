package relay

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func TestClientSend(t *testing.T) {
	const secret = "some-secret-string"
	const filename = "file.txt"
	const contents = "file contents"

	var request1 *http.Request
	var request2 *http.Request

	file := ioutil.NopCloser(strings.NewReader(contents))
	sent := bytes.NewBuffer([]byte{})

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost:
			request1 = r
			if _, err := io.Copy(w, strings.NewReader(secret+"\n")); err != nil {
				t.Error("sending secret:", err)
			}
		case http.MethodPut:
			request2 = r
			if _, err := io.Copy(sent, r.Body); err != nil {
				t.Error("copying file:", err)
			}
		}
	}))

	u, _ := url.Parse(server.URL)
	client := NewClient(u.Host)

	sec, send, err := client.Offer(filename, file)
	if err != nil {
		t.Error("client.Offer:", err)
	}

	t.Run("POSTs to /file", func(t *testing.T) {
		got := request1.Method + " " + request1.URL.Path
		want := "POST /file"

		if got != want {
			t.Errorf("got %q, want %q", got, want)
		}
	})

	t.Run("suggests filename", func(t *testing.T) {
		got := request1.Header.Get(filenameHeader)
		want := filename

		if got != want {
			t.Errorf("got %q, want %q", got, want)
		}
	})

	t.Run("returns secret", func(t *testing.T) {
		got := sec
		want := secret

		if got != want {
			t.Errorf("got %q, want %q", got, want)
		}
	})

	t.Run("waits to stream", func(t *testing.T) {
		got := sent.Len()
		want := 0

		if got != want {
			t.Errorf("got %v, want %v", got, want)
		}
	})

	if err := send(); err != nil {
		t.Errorf("send error: %w", err)
	}

	t.Run("PUTs to /file/{secret}", func(t *testing.T) {
		got := request2.Method + " " + request2.URL.Path
		want := "PUT /file/" + secret

		if got != want {
			t.Errorf("got %q, want %q", got, want)
		}
	})

	t.Run("streams file", func(t *testing.T) {
		got := sent.String()
		want := contents

		if got != want {
			t.Errorf("got %q, want %q", got, want)
		}
	})
}

func TestClientReceive(t *testing.T) {
	const secret = "some-secret-string"
	const filename = "file.txt"
	const contents = "file contents"

	file := ioutil.NopCloser(strings.NewReader(contents))

	var request *http.Request

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		request = r
		w.Header().Add(filenameHeader, filename)
		if _, err := io.Copy(w, file); err != nil {
			t.Errorf("copying file: %w", err)
		}
	}))

	u, _ := url.Parse(server.URL)
	client := NewClient(u.Host)

	suggestedName, stream, err := client.Receive(secret)
	if err != nil {
		t.Errorf("receiving: %w", err)
	}

	received, err := ioutil.ReadAll(stream)
	if err != nil {
		t.Errorf("reading stream: %w", err)
	}

	t.Run("requests GET /file/{secret}", func(t *testing.T) {
		got := request.Method + " " + request.URL.Path
		want := "GET /file/" + secret

		if got != want {
			t.Errorf("got %q, want %q", got, want)
		}
	})

	t.Run("returns suggested filename", func(t *testing.T) {
		got := suggestedName
		want := filename

		if got != want {
			t.Errorf("got %q, want %q", got, want)
		}
	})

	t.Run("streams file", func(t *testing.T) {
		got := fmt.Sprintf("%s", received)
		want := contents

		if got != want {
			t.Errorf("got %q, want %q", got, want)
		}
	})
}
