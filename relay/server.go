package relay

import (
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"strings"
	"sync"
)

const (
	filenameHeader = "suggested-filename"
	secretHeader   = "secret"
)

type secret string

type offer struct {
	filename string
	receiver chan http.ResponseWriter
	done     chan struct{}
}

type Server struct {
	http.Server
	rng    *rand.Rand
	router *http.ServeMux

	sync.RWMutex
	offers map[secret]offer
}

func NewServer(addr string, rng *rand.Rand) (*Server, error) {
	certs, err := certificates()
	if err != nil {
		return nil, fmt.Errorf("loading certs: %w", err)
	}

	s := &Server{
		Server: http.Server{
			Addr: addr,
			TLSConfig: &tls.Config{
				Certificates: certs,
			},
		},
		rng:    rng,
		router: http.NewServeMux(),
		offers: make(map[secret]offer),
	}
	s.router.Handle("/send", s.handleSend())
	s.router.Handle("/receive", s.handleReceive())
	s.Handler = s.router

	return s, nil
}

// TODO: timeout
func (s *Server) handleSend() http.HandlerFunc {
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
		off, sec, err := s.createOffer(filename)
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
func (s *Server) handleReceive() http.HandlerFunc {
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

		sec := secret(r.Header.Get(secretHeader))
		off, err := s.findOffer(sec)
		if err != nil {
			// TODO: logging
			w.WriteHeader(http.StatusNotFound)
			return
		}

		w.Header().Add(filenameHeader, off.filename)
		response.Flush()

		off.receiver <- w // s.handleSend now writes to w
		<-off.done        // wait until s.handleSend is complete
	}
}

func (s *Server) createOffer(filename string) (offer, secret, error) {
	s.Lock()
	defer s.Unlock()

	var sec secret
	for exists := true; exists; sec = s.nextSecret() {
		_, exists = s.offers[sec]
	}

	off := offer{
		receiver: make(chan http.ResponseWriter),
		filename: fmt.Sprintf("%s", filename),
		done:     make(chan struct{}),
	}
	s.offers[sec] = off

	return off, sec, nil
}

func (s *Server) findOffer(sec secret) (offer, error) {
	s.Lock()
	defer s.Unlock()

	off, ok := s.offers[sec]
	if !ok {
		return off, errors.New("no such offer or secret")
	}
	return off, nil
}

func (s *Server) nextSecret() secret {
	n := len(words)
	a, b, c := s.rng.Intn(n), s.rng.Intn(n), s.rng.Intn(n)
	return secret(words[a] + "-" + words[b] + "-" + words[c])
}
