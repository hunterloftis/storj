package relay

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"
)

const (
	filenameHeader = "suggested-filename"
	offerTimeout   = 10 * time.Minute
)

type offer struct {
	filename string
	address  string
	receiver chan http.ResponseWriter
	ctx      context.Context
	cancel   context.CancelFunc
}

// Handler is the HTTP request handler that relays messages between clients.
type Handler struct {
	router  *http.ServeMux
	secrets fmt.Stringer
	logger  io.Writer

	sync.RWMutex
	offers map[string]offer
}

// NewHandler returns a new Handler.
//
// It generates secret strings via the provided Stringer and logs events to the provided Writer.
func NewHandler(secrets fmt.Stringer, logger io.Writer) *Handler {
	h := &Handler{
		secrets: secrets,
		offers:  make(map[string]offer),
		router:  http.NewServeMux(),
		logger:  logger,
	}

	h.router.Handle("/file", h.handleNew())
	h.router.Handle("/file/", h.handleExisting())

	return h
}

// ServeHTTP allows the handler to serve HTTP requests.
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h.router.ServeHTTP(w, r)
}

func (h *Handler) handleNew() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusNotFound)
			return
		}

		filename := r.Header.Get(filenameHeader)
		secret, err := h.createOffer(filename, r.RemoteAddr)
		if err != nil {
			fmt.Fprintln(h.logger, fmt.Errorf("creating offer: %w", err))
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		if _, err := fmt.Fprintln(w, secret); err != nil {
			fmt.Fprintln(h.logger, fmt.Errorf("sending secret: %w", err))
			return
		}
	}
}

func (h *Handler) handleExisting() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		split := strings.Split(r.URL.Path[1:], "/")
		if len(split) < 2 {
			w.WriteHeader(http.StatusNotFound)
			return
		}

		secret := split[1]
		off, err := h.findOffer(secret)
		if err != nil {
			fmt.Fprintln(h.logger, fmt.Errorf("finding offer: %w", err))
			w.WriteHeader(http.StatusNotFound)
			return
		}

		switch r.Method {
		case http.MethodPut:
			h.handleSend(w, r, off)
		case http.MethodGet:
			h.handleReceive(w, r, off)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}
}

func (h *Handler) handleSend(w http.ResponseWriter, r *http.Request, off offer) {
	if off.address != r.RemoteAddr {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	// wait for a receiver to connect
	select {
	case receiveWriter := <-off.receiver:
		defer off.cancel()
		if _, err := io.Copy(receiveWriter, r.Body); err != nil {
			fmt.Fprintln(h.logger, fmt.Errorf("sending file: %w", err))
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

	case <-off.ctx.Done():
		w.WriteHeader(http.StatusRequestTimeout)
		return
	}
}

func (h *Handler) handleReceive(w http.ResponseWriter, r *http.Request, off offer) {
	w.Header().Add(filenameHeader, off.filename)
	off.receiver <- w
	<-off.ctx.Done() // wait until h.handleSend is complete
}

func (h *Handler) createOffer(filename, address string) (secret string, err error) {
	h.Lock()
	defer h.Unlock()

	ctx, cancel := context.WithTimeout(context.Background(), offerTimeout)

	off := offer{
		filename: filename,
		address:  address,
		receiver: make(chan http.ResponseWriter),
		ctx:      ctx,
		cancel:   cancel,
	}

	// ensure secret is unique
	for exists := true; exists; {
		secret = h.secrets.String()
		_, exists = h.offers[secret]
	}

	h.offers[secret] = off

	// destroy the offer once it's completed
	go func() {
		<-off.ctx.Done()
		h.Lock()
		defer h.Unlock()
		delete(h.offers, secret)
	}()

	return secret, nil
}

func (h *Handler) findOffer(secret string) (offer, error) {
	h.Lock()
	defer h.Unlock()

	off, ok := h.offers[secret]
	if !ok {
		return offer{}, fmt.Errorf("no such secret: %v", secret)
	}

	return off, nil
}
