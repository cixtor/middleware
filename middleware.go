package middlware

import (
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
