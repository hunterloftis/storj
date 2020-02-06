package relay

import (
	"bufio"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"runtime"
	"strings"
	"testing"
)

// test utilities

const (
	filename = "filename.txt"
	contents = "file contents"
	secret   = "some-secret-string"
)

type secretList struct {
	strings []string
	i       int
}

func newSecretList(strings ...string) *secretList {
	return &secretList{
		strings: strings,
	}
}

func (sl *secretList) String() (secret string) {
	secret = sl.strings[sl.i%len(sl.strings)]
	sl.i++
	return
}

type countWriter struct {
	*httptest.ResponseRecorder
	bytes int
}

func newCountWriter() *countWriter {
	return &countWriter{
		ResponseRecorder: httptest.NewRecorder(),
	}
}

func (cw *countWriter) Write(b []byte) (int, error) {
	n := len(b)
	cw.bytes += n
	return n, nil
}

func (cw *countWriter) WriteString(str string) (int, error) {
	n := len(str)
	cw.bytes += n
	return n, nil
}

type genReader struct {
	remaining int
}

func (gr *genReader) Read(p []byte) (int, error) {
	if gr.remaining <= 0 {
		return 0, io.EOF
	}
	n := len(p)
	if n > 1000 {
		n = 1000
	}
	if n > gr.remaining {
		n = gr.remaining
	}
	for i := 0; i < n; i++ {
		p[i] = 'X'
	}
	gr.remaining -= n
	return n, nil
}

// tests

func TestRelayGoldenPath(t *testing.T) {
	handler := NewHandler(newSecretList(secret))

	t.Run("returns a secret code", func(t *testing.T) {
		file := strings.NewReader(contents)
		request, _ := http.NewRequest(http.MethodPost, "/send", file)
		request.Header.Set(filenameHeader, filename)
		w := httptest.NewRecorder()

		go handler.ServeHTTP(w, request)
		for w.Body.Len() == 0 {
		}

		resp := w.Result()

		got, err := bufio.NewReader(resp.Body).ReadString('\n')
		if err != nil {
			t.Error("reading secret:", err)
		}
		want := secret + "\n"

		if got != want {
			t.Errorf("got %q, want %q", got, want)
		}
	})

	t.Run("receives a file", func(t *testing.T) {
		request, _ := http.NewRequest(http.MethodGet, "/receive", nil)
		request.Header.Set(secretHeader, secret)
		w := httptest.NewRecorder()

		handler.ServeHTTP(w, request)
		resp := w.Result()
		body, _ := ioutil.ReadAll(resp.Body)

		t.Run("contents match", func(t *testing.T) {
			got := fmt.Sprintf("%s", body)
			want := contents

			if got != want {
				t.Errorf("got %q, want %q", got, want)
			}
		})

		t.Run("filenames match", func(t *testing.T) {
			got := resp.Header.Get(filenameHeader)
			want := filename

			if got != want {
				t.Errorf("got %q, want %q", got, want)
			}
		})
	})
}

func TestRelayLargeFile(t *testing.T) {
	const size = 1000 * 1000 * 1000 // 1 GB
	var start, end runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&start)
	handler := NewHandler(newSecretList(secret))

	{
		file := &genReader{size}
		request, _ := http.NewRequest(http.MethodPost, "/send", file)
		request.Header.Set(filenameHeader, filename)
		w := httptest.NewRecorder()
		go handler.ServeHTTP(w, request)
		for w.Body.Len() == 0 {
		}
	}

	t.Run("transfers 1 GB of data", func(t *testing.T) {
		request, _ := http.NewRequest(http.MethodGet, "/receive", nil)
		request.Header.Set(secretHeader, secret)
		writer := newCountWriter()
		handler.ServeHTTP(writer, request)

		got := writer.bytes
		want := size

		if got != want {
			t.Errorf("sent %v bytes, want %v", got, want)
		}
	})

	t.Run("never uses > 4 MB", func(t *testing.T) {
		const limit = 4 * 1000 * 1000

		runtime.ReadMemStats(&end)
		alloc := end.TotalAlloc - start.TotalAlloc
		if alloc > limit {
			t.Errorf("used %v bytes, limit %v", alloc, limit)
		}
	})
}

func TestRelaySimultaneous(t *testing.T) {
	secrets := []string{"a-a-a", "b-b-b", "c-c-c"}
	handler := NewHandler(newSecretList(secrets...))

	for i := 0; i < len(secrets); i++ {
		file := strings.NewReader(fmt.Sprintf("file contents %v", i))
		request, _ := http.NewRequest(http.MethodPost, "/send", file)
		request.Header.Set(filenameHeader, fmt.Sprintf("file-%v.txt", i))
		w := httptest.NewRecorder()
		go handler.ServeHTTP(w, request)
		for w.Body.Len() == 0 {
		}
	}

	for i := 0; i < len(secrets); i++ {
		request, _ := http.NewRequest(http.MethodGet, "/receive", nil)
		request.Header.Set(secretHeader, secrets[i])
		w := httptest.NewRecorder()

		handler.ServeHTTP(w, request)
		resp := w.Result()
		body, _ := ioutil.ReadAll(resp.Body)

		got := fmt.Sprintf("%s", body)
		want := fmt.Sprintf("file contents %v", i)

		if got != want {
			t.Errorf("got %q, want %q", got, want)
		}

		got = resp.Header.Get(filenameHeader)
		want = fmt.Sprintf("file-%v.txt", i)

		if got != want {
			t.Errorf("got filename %q, want %q", got, want)
		}
	}
}

func TestRelayWrongSecret(t *testing.T) {
	handler := NewHandler(newSecretList(secret))

	{
		file := strings.NewReader(contents)
		request, _ := http.NewRequest(http.MethodPost, "/send", file)
		request.Header.Set(filenameHeader, filename)
		w := httptest.NewRecorder()

		go handler.ServeHTTP(w, request)
		for w.Body.Len() == 0 {
		}
	}

	request, _ := http.NewRequest(http.MethodGet, "/receive", nil)
	request.Header.Set(secretHeader, "wrong-secret")
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, request)
	resp := w.Result()
	body, _ := ioutil.ReadAll(resp.Body)

	t.Run("returns a 404", func(t *testing.T) {
		got := resp.StatusCode
		want := http.StatusNotFound

		if got != want {
			t.Errorf("got %v, want %v", got, want)
		}
	})

	t.Run("serves no content", func(t *testing.T) {
		got := fmt.Sprintf("%s", body)
		want := ""

		if got != want {
			t.Errorf("got %q, want %q", got, want)
		}
	})
}

func TestRelayWrongMethod(t *testing.T) {
	handler := NewHandler(newSecretList(secret))

	t.Run("GET to /send fails", func(t *testing.T) {
		file := strings.NewReader(contents)
		request, _ := http.NewRequest(http.MethodGet, "/send", file)
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, request)
		resp := w.Result()

		got := resp.StatusCode
		want := http.StatusMethodNotAllowed

		if got != want {
			t.Errorf("got %v, want %v", got, want)
		}
	})

	t.Run("POST to /receive fails", func(t *testing.T) {
		request, _ := http.NewRequest(http.MethodPost, "/receive", nil)
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, request)
		resp := w.Result()

		got := resp.StatusCode
		want := http.StatusMethodNotAllowed

		if got != want {
			t.Errorf("got %v, want %v", got, want)
		}
	})
}
