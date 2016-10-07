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

func (m *Middleware) Handle(method, path string, handle http.HandlerFunc) {
	var node Node
	var parts []string
	var usable []string

	node.Path = "/"
	node.Dispatcher = handle
	parts = strings.Split(path, "/")

	// Separate dynamic parameters from the static URL.
	for _, section := range parts {
		if section == "" {
			continue
		}

		if len(section) > 1 && section[0] == ':' {
			node.Params = append(node.Params, section[1:])
			node.NumSections += 1
			node.NumParams += 1
			continue
		}

		usable = append(usable, section)
		node.NumSections += 1
	}

	node.Path += strings.Join(usable, "/")

	m.Nodes[method] = append(m.Nodes[method], &node)
}
