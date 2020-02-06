package relay

import (
	"bytes"
	"crypto/tls"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"golang.org/x/net/http2"
)

func TestClientSend(t *testing.T) {
	const secret = "some-secret-string"
	const filename = "file.txt"
	const contents = "file contents"

	var requested string
	var suggestedFilename string

	file := ioutil.NopCloser(strings.NewReader("file contents"))
	sent := bytes.NewBuffer([]byte{})
	blocker := make(chan struct{})

	ts := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requested = r.Method + " " + r.URL.Path
		response, _ := w.(http.Flusher)
		suggestedFilename = r.Header.Get(filenameHeader)
		if _, err := io.Copy(w, strings.NewReader(secret+"\n")); err != nil {
			t.Error("sending secret:", err)
		}
		response.Flush()

		<-blocker

		if _, err := io.Copy(sent, r.Body); err != nil {
			t.Error("copying file:", err)
		}
	}))

	ts.TLS = &tls.Config{
		CipherSuites: []uint16{tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256},
		NextProtos:   []string{http2.NextProtoTLS},
	}
	ts.StartTLS()

	u, _ := url.Parse(ts.URL)
	client := NewClient(u.Host, true)

	sec, wait, err := client.Send(filename, file)
	if err != nil {
		t.Error("client.Send:", err)
	}

	t.Run("posts to /send", func(t *testing.T) {
		got := requested
		want := "POST /send"

		if got != want {
			t.Errorf("got %q, want %q", got, want)
		}
	})

	t.Run("suggests filename", func(t *testing.T) {
		got := suggestedFilename
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
		got := fmt.Sprintf("%s", sent)
		want := ""

		if got != want {
			t.Errorf("got %q, want %q", got, want)
		}
	})

	close(blocker)
	wait()

	t.Run("streams file", func(t *testing.T) {
		got := fmt.Sprintf("%s", sent)
		want := contents

		if got != want {
			t.Errorf("got %q, want %q", got, want)
		}
	})
}

func TestClientReceive(t *testing.T) {

}
