package middleware

import (
	"log"
	"net/http"
	"os"
	"time"
)

// defaultPort is the TCP port number to attach the server.
const defaultPort = "8080"

// defaultHost is the IP address to attach the web server.
const defaultHost = "0.0.0.0"

// defaultShutdownTimeout is the maximum time before server halt.
const defaultShutdownTimeout = 5 * time.Second

// Middleware is the base of the library and the entry point for every HTTP
// request. It acts as a modular interface that wraps around http.Handler to
// add additional functionality like custom routes, separated HTTP method
// processors and named parameters.
type Middleware struct {
	Host              string
	Port              string
	NotFound          http.Handler
	IdleTimeout       time.Duration
	ReadTimeout       time.Duration
	WriteTimeout      time.Duration
	ShutdownTimeout   time.Duration
	ReadHeaderTimeout time.Duration
	nodes             map[string][]*route
	logger            *log.Logger
	serverInstance    *http.Server
	serverShutdown    chan bool
	allowedAddresses  []string
	deniedAddresses   []string
	restrictionType   string
}

// route is a data structure to keep the defined routes, named parameters and
// HTTP handler. Some routes like the document root and static files might set
// another property to force the ServeHTTP method to return immediately for
// every match in the URL no matter if the named parameters do not match.
type route struct {
	path            string
	params          []string
	numParams       int
	numSections     int
	dispatcher      http.HandlerFunc
	isStaticHandler bool
}

// httpParam represents a single parameter in the URL.
type httpParam struct {
	Name  string
	Value string
}

// contextKey is the key for the parameters in the request Context.
type contextKey string

// paramsKey is the key for the parameters in the request Context.
var paramsKey = contextKey("MiddlewareParameter")

// New returns a new initialized Middleware.
func New() *Middleware {
	m := new(Middleware)

	m.nodes = make(map[string][]*route)
	m.logger = log.New(os.Stdout, "", log.LstdFlags)

	return m
}

// ServeHTTP dispatches the request to the handler whose pattern most closely
// matches the request URL. Additional to the standard functionality this also
// logs every direct HTTP request into the standard output.
func (m *Middleware) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	var query string

	start := time.Now()
	writer := response{w, 0, 0}

	if r.URL.RawQuery != "" {
		query = "?" + r.URL.RawQuery
	}

	m.handleRequest(&writer, r)

	m.logger.Printf(
		"%s %s \"%s %s %s\" %d %d \"%s\" %v",
		r.Host,
		r.RemoteAddr,
		r.Method,
		r.URL.Path+query,
		r.Proto,
		writer.Status,
		writer.Length,
		r.Header.Get("User-Agent"),
		time.Now().Sub(start),
	)
}

// ServeFiles serves files from the given file system root.
//
// A pre-check is executed to prevent directory listing attacks.
func (m *Middleware) ServeFiles(root string, prefix string) http.HandlerFunc {
	fs := http.FileServer(http.Dir(root))
	handler := http.StripPrefix(prefix, fs)

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path[len(r.URL.Path)-1] == '/' {
			http.Error(w, http.StatusText(403), http.StatusForbidden)
			return
		}

		handler.ServeHTTP(w, r)
	})
}
