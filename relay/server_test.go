package relay

import (
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

func TestHandlerGoldenPath(t *testing.T) {
	const filename = "filename.txt"
	const contents = "file contents"
	const secret = "some-secret-string"

	handler := NewHandler(newSecretList(secret), ioutil.Discard)

	t.Run("POST returns a secret code", func(t *testing.T) {
		request, _ := http.NewRequest(http.MethodPost, "/file", nil)
		request.Header.Set(filenameHeader, filename)
		w := httptest.NewRecorder()

		handler.ServeHTTP(w, request)
		resp := w.Result()

		got, _ := ioutil.ReadAll(resp.Body)
		want := secret + "\n"

		if string(got) != want {
			t.Errorf("got %q, want %q", got, want)
		}
	})

	go func() {
		file := strings.NewReader(contents)
		request, _ := http.NewRequest(http.MethodPut, "/file/"+secret, file)
		w := httptest.NewRecorder()

		handler.ServeHTTP(w, request)
	}()

	t.Run("GET receives a file", func(t *testing.T) {
		request, _ := http.NewRequest(http.MethodGet, "/file/"+secret, nil)
		w := httptest.NewRecorder()

		handler.ServeHTTP(w, request)
		resp := w.Result()
		body, _ := ioutil.ReadAll(resp.Body)

		t.Run("contents match", func(t *testing.T) {
			got := string(body)
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

func TestHandlerLargeFile(t *testing.T) {
	const filename = "filename.txt"
	const contents = "file contents"
	const secret = "some-secret-string"
	const size = 1000 * 1000 * 1000 // 1 GB
	var start, end runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&start)
	handler := NewHandler(newSecretList(secret), ioutil.Discard)

	offered := make(chan struct{})

	go func() {
		request, _ := http.NewRequest(http.MethodPost, "/file", nil)
		request.Header.Set(filenameHeader, filename)
		w := newCountWriter()
		handler.ServeHTTP(w, request)

		close(offered)

		file := &genReader{size}
		req2, _ := http.NewRequest(http.MethodPut, "/file/"+secret, file)
		w2 := newCountWriter()
		handler.ServeHTTP(w2, req2)
	}()

	<-offered

	t.Run("receives 1 GB of data", func(t *testing.T) {
		request, _ := http.NewRequest(http.MethodGet, "/file/"+secret, nil)
		writer := newCountWriter()
		handler.ServeHTTP(writer, request)

		got := writer.bytes
		want := size

		if got != want {
			t.Errorf("received %v bytes, want %v", got, want)
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

func TestHandlerSimultaneous(t *testing.T) {
	secrets := []string{"a-a-a", "b-b-b", "c-c-c"}
	handler := NewHandler(newSecretList(secrets...), ioutil.Discard)

	for i := 0; i < len(secrets); i++ {
		request, _ := http.NewRequest(http.MethodPost, "/file", nil)
		request.Header.Set(filenameHeader, fmt.Sprintf("file-%v.txt", i))
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, request)

		file := strings.NewReader(fmt.Sprintf("file contents %v", i))
		req2, _ := http.NewRequest(http.MethodPut, "/file/"+secrets[i], file)
		w2 := httptest.NewRecorder()
		go handler.ServeHTTP(w2, req2)
	}

	for i := 0; i < len(secrets); i++ {
		request, _ := http.NewRequest(http.MethodGet, "/file/"+secrets[i], nil)
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

func TestHandlerWrongSecret(t *testing.T) {
	const filename = "filename.txt"
	const contents = "file contents"
	const secret = "some-secret-string"
	handler := NewHandler(newSecretList(secret), ioutil.Discard)

	offered := make(chan struct{})

	go func() {
		request, _ := http.NewRequest(http.MethodPost, "/file", nil)
		request.Header.Set(filenameHeader, filename)
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, request)

		close(offered)

		file := strings.NewReader(contents)
		req2, _ := http.NewRequest(http.MethodPut, "/file/"+secret, file)
		w2 := httptest.NewRecorder()
		handler.ServeHTTP(w2, req2)
	}()

	<-offered

	request, _ := http.NewRequest(http.MethodGet, "/file/wrong-secret", nil)
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
