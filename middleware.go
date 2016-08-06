package middlware

import (
	"log"
	"net/http"
	"time"
)

const DEFAULT_PORT = "8080"

type Middleware struct {
	Port         string
	Nodes        map[string][]*Node
	NotFound     http.Handler
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
}

type StatusWriter struct {
	http.ResponseWriter
	Status int
	Length int
}

type Node struct {
	Path            string
	Params          []string
	NumParams       int
	NumSections     int
	Dispatcher      http.HandlerFunc
	MatchEverything bool
}

func New() *Middleware {
	return &Middleware{Nodes: make(map[string][]*Node)}
}

func (w *StatusWriter) WriteHeader(status int) {
	w.Status = status
	w.ResponseWriter.WriteHeader(status)
}

func (w *StatusWriter) Write(b []byte) (int, error) {
	if w.Status == 0 {
		w.Status = 200
	}

	w.Length = len(b)

	return w.ResponseWriter.Write(b)
}

func (m *Middleware) ListenAndServe() {
	if m.Port == "" {
		m.Port = DEFAULT_PORT
	}

	address := "127.0.0.1:" + m.Port
	server := &http.Server{
		Addr:         address,
		Handler:      m, /* http.DefaultServeMux */
		ReadTimeout:  m.ReadTimeout * time.Second,
		WriteTimeout: m.WriteTimeout * time.Second,
	}

	log.Println("Running server on", address)
	log.Println("PANIC:", server.ListenAndServe())
}
