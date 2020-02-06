package relay

import (
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
)

const (
	filenameHeader = "suggested-filename"
	secretHeader   = "secret"
)

type offer struct {
	filename string
	receiver chan http.ResponseWriter
	done     chan struct{}
}

func NewServer(addr string, handler *Handler) (*http.Server, error) {
	cert, err := tls.X509KeyPair(cert, certKey)
	if err != nil {
		return nil, fmt.Errorf("loading certs: %w", err)
	}

	s := &http.Server{
		Addr: addr,
		TLSConfig: &tls.Config{
			Certificates: []tls.Certificate{cert},
		},
		Handler: handler,
	}
	return s, nil
}

type Handler struct {
	router  *http.ServeMux
	secrets fmt.Stringer

	sync.RWMutex
	offers map[string]offer
}

func NewHandler(secrets fmt.Stringer) *Handler {
	h := &Handler{
		secrets: secrets,
		router:  http.NewServeMux(),
		offers:  make(map[string]offer),
	}
	h.router.Handle("/send", h.handleSend())
	h.router.Handle("/receive", h.handleReceive())

	return h
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h.router.ServeHTTP(w, r)
}

// TODO: timeout
func (h *Handler) handleSend() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}

		response, ok := w.(http.Flusher)
		if !ok {
			// TODO: logging
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		filename := r.Header.Get(filenameHeader)
		off, sec, err := h.createOffer(filename)
		if err != nil {
			// TODO: logging
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		if _, err := io.Copy(w, strings.NewReader(string(sec)+"\n")); err != nil {
			// TODO: logging
			return
		}
		response.Flush()

		// wait for a matching receiver to connect
		receiveWriter := <-off.receiver

		if _, err := io.Copy(receiveWriter, r.Body); err != nil {
			// TODO: logging
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		close(off.done)
	}
}

// TODO: timeout
func (h *Handler) handleReceive() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}

		response, ok := w.(http.Flusher)
		if !ok {
			// TODO: logging
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		secret := r.Header.Get(secretHeader)
		off, err := h.findOffer(secret)
		if err != nil {
			// TODO: logging
			w.WriteHeader(http.StatusNotFound)
			return
		}

		w.Header().Add(filenameHeader, off.filename)
		response.Flush()

		off.receiver <- w // h.handleSend now writes to w
		<-off.done        // wait until h.handleSend is complete
	}
}

func (h *Handler) createOffer(filename string) (off offer, secret string, err error) {
	h.Lock()
	defer h.Unlock()

	for exists := true; exists; {
		secret = h.secrets.String()
		_, exists = h.offers[secret]
	}

	h.offers[secret] = offer{
		receiver: make(chan http.ResponseWriter),
		filename: fmt.Sprintf("%s", filename),
		done:     make(chan struct{}),
	}

	return h.offers[secret], secret, nil
}

func (h *Handler) findOffer(secret string) (offer, error) {
	h.Lock()
	defer h.Unlock()

	off, ok := h.offers[secret]
	if !ok {
		return off, errors.New("no such offer or secret")
	}
	return off, nil
}
