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
