package middleware

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"path"
	"strings"
	"time"
)

// defaultPort is the TCP port number to attach the server.
const defaultPort = "8080"

// defaultHost is the IP address to attach the web server.
const defaultHost = "0.0.0.0"

// defaultShutdownTimeout is the maximum time before server halt.
const defaultShutdownTimeout = 5 * time.Second

// allowAccessExcept is the ID for the "allow" restriction rule.
const allowAccessExcept = 0x6411a9

// denyAccessExcept is the ID for the "deny" restriction rule.
const denyAccessExcept = 0x32afb2

// errNoMatch represents a simple matching error.
var errNoMatch = errors.New("no matching route")

// Middleware is the base of the library and the entry point for every HTTP
// request. It acts as a modular interface that wraps around http.Handler to
// add additional functionality like custom routes, separated HTTP method
// processors and named parameters.
type Middleware struct {
	Host              string
	Port              string
	Logger            *log.Logger
	NotFound          http.Handler
	IdleTimeout       time.Duration
	ReadTimeout       time.Duration
	WriteTimeout      time.Duration
	ShutdownTimeout   time.Duration
	ReadHeaderTimeout time.Duration
	chain             func(http.Handler) http.Handler
	nodes             map[string][]route
	serverInstance    *http.Server
	serverShutdown    chan bool
	allowedAddresses  []string
	deniedAddresses   []string
	restrictionType   int
}

// route is a data structure to keep the defined routes, named parameters and
// HTTP handler. Some routes like the document root and static files might set
// another property to force the ServeHTTP method to return immediately for
// every match in the URL no matter if the named parameters do not match.
type route struct {
	// path is the raw URL: `/lorem/:ipsum/dolor`
	path string
	// parts is a list of sections representing the URL.
	parts []rpart
	// glob is true if the route has a global catcher.
	glob bool
	// dispatcher is the HTTP handler function for the route.
	dispatcher http.HandlerFunc
}

// rpart represents each part of the route.
//
// Example:
//
//   /lorem/:ipsum/dolor -> []section{
//     section{name:"<root>", dyna: false, root: true},
//     section{name:"lorem",  dyna: false, root: false},
//     section{name:":ipsum", dyna: true,  root: false},
//     section{name:"dolor",  dyna: false, root: false},
//   }
type rpart struct {
	// name is the raw text in the URL.
	name string
	// dyna is short for “dynamic”; true if `/^:\S+/` false otherwise
	dyna bool
	// root is true if the route part is the first in the list.
	root bool
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
//
// By default, the HTTP response logger is enabled, and the text is written to
// `/dev/stdout`. You can disable this by setting `m.Logger = nil` where “m” is
// an instance of `middleware.New()`. You can also writes the logs to a buffer
// or any other Go logger interface defined as `log.New()`.
func New() *Middleware {
	m := new(Middleware)

	m.nodes = make(map[string][]route)
	m.Logger = log.New(os.Stdout, "", log.LstdFlags)

	return m
}

// compose follows the HTTP handler chain to execute additional middlewares.
func compose(f, g func(http.Handler) http.Handler) func(http.Handler) http.Handler {
	return func(h http.Handler) http.Handler {
		return g(f(h))
	}
}

// Use adds a middleware to the global middleware chain.
//
// The additional middlewares are executed in the same order as they are added
// to the chain. For example, if you have wrappers to add security headers, a
// session management system, and a file system cache policy, you can attach
// them to the main router like this:
//
//   router.Use(headersMiddleware)
//   router.Use(sessionMiddleware)
//   router.Use(filesysMiddleware)
//
// They will run as follows:
//
//   headersMiddleware(
//     sessionMiddleware(
//       filesysMiddleware(
//         func(http.ResponseWriter, *http.Request)
//       )
//     )
//   )
//
// Use the following template to create more middlewares:
//
//   func foobar(next http.Handler) http.Handler {
//       return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
//           […]
//           next.ServeHTTP(w, r)
//       })
//   }
func (m *Middleware) Use(f func(http.Handler) http.Handler) {
	if m.chain == nil {
		m.chain = f
		return
	}

	m.chain = compose(f, m.chain)
}

// ServeHTTP dispatches the request to the handler whose pattern most closely
// matches the request URL. Additional to the standard functionality this also
// logs every direct HTTP request into the standard output.
func (m *Middleware) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if m.Logger == nil {
		m.handleRequest(w, r)
		return
	}

	var query string

	start := time.Now()
	writer := response{w, 0, 0}

	if r.URL.RawQuery != "" {
		query = "?" + r.URL.RawQuery
	}

	m.handleRequest(&writer, r)

	m.Logger.Printf(
		"%s %s \"%s %s %s\" %d %d \"%s\" %v",
		r.Host,
		r.RemoteAddr,
		r.Method,
		r.URL.Path+query,
		r.Proto,
		writer.Status,
		writer.Length,
		r.Header.Get("User-Agent"),
		time.Since(start),
	)
}

// ServeFiles serves files from the given file system root.
//
// A pre-check is executed to prevent directory listing attacks.
func (m *Middleware) ServeFiles(root string, prefix string) http.HandlerFunc {
	fs := http.FileServer(http.Dir(root))
	handler := http.StripPrefix(prefix, fs)

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var err error
		var raw string
		var fifo os.FileInfo

		// convert URL into file system path.
		raw = root + r.URL.Path[len(prefix):]

		if fifo, err = os.Stat(raw); err != nil {
			// requested resource does not exists; return 404 Not Found
			http.Error(w, http.StatusText(404), http.StatusNotFound)
			return
		}

		if fifo.IsDir() {
			// requested resource is a directory; return 403 Forbidden
			http.Error(w, http.StatusText(403), http.StatusForbidden)
			return
		}

		handler.ServeHTTP(w, r)
	})
}

// handleRequest responds to an HTTP request.
//
// The function selects the HTTP handler by traversing a tree that contains a
// list of all the defined URLs without the dynamic parameters (if any). If the
// defined URL doesn’t contains dynamic parameters, the function executes the
// HTTP handler immediately if the URL path matches the request. If there are
// dynamic parameters, the function checks if the URL contains enough data to
// extract them, if there is not enough data, it responds with “404 Not Found“,
// otherwise, it attaches the values for the corresponding parameters to the
// request context, then executes the HTTP handler.
//
// Here is an example of a successful request:
//
//   Defined URL: /foo/bar/:group
//   Request URL: /foo/bar/example
//
// This request returns a "200 OK" and the HTTP handler can then obtain a copy
// of the value for the “group” parameter using `middleware.Param()`. Or simply
// by reading the raw parameter from the request context.
//
// Here is an example of an invalid request:
//
//   Defined URL: /foo/bar/:group
//   Request URL: /foo/bar/
//   Request URL: /foo/bar
//   Request URL: /foo/
//   Request URL: /foo
//   Request URL: /
//
// All these requests will return “404 Not Found” because none of them matches
// the defined URL. This is because trailing slashes are ignored, so even the
// first attempt (which is similar to what the HTTP handler is expecting) will
// fail as there is not enough data to set the value for the “group” parameter.
func (m *Middleware) handleRequest(w http.ResponseWriter, r *http.Request) {
	children, ok := m.nodes[r.Method]

	if !ok {
		// HTTP method not allowed, return “405 Method Not Allowed”.
		http.Error(w, http.StatusText(405), http.StatusMethodNotAllowed)
		return
	}

	if r.URL.Path == "" || r.URL.Path[0] != '/' {
		// URL prefix is invalid, return “400 Bad Request”.
		http.Error(w, http.StatusText(400), http.StatusBadRequest)
		return
	}

	if m.restrictionType == allowAccessExcept && inArray(m.deniedAddresses, remoteAddr(r)) {
		// IP address was blacklisted, return “403 Forbidden”.
		http.Error(w, http.StatusText(403), http.StatusForbidden)
		return
	}

	if m.restrictionType == denyAccessExcept && !inArray(m.allowedAddresses, remoteAddr(r)) {
		// IP address is not whitelisted, return “403 Forbidden”.
		http.Error(w, http.StatusText(403), http.StatusForbidden)
		return
	}

	// iterate against the routes to find a handler.
	child, params, err := m.findHandler(r, children)

	// send “404 Not Found” if there is no handler.
	if err != nil || child.dispatcher == nil {
		if m.NotFound != nil {
			m.NotFound.ServeHTTP(w, r)
			return
		}

		http.NotFound(w, r)
		return
	}

	if len(params) > 0 {
		// save params in the request context.
		r = r.WithContext(context.WithValue(
			r.Context(),
			paramsKey,
			params,
		))
	}

	child.dispatcher(w, r)
}

// findHandler
func (m *Middleware) findHandler(r *http.Request, children []route) (route, []httpParam, error) {
	for _, child := range children {
		// side-by-side match; no params.
		if r.URL.Path == child.path {
			return child, []httpParam{}, nil
		}

		// global match; match everything with the same prefix.
		if child.glob && strings.HasPrefix(r.URL.Path, child.path) {
			return child, []httpParam{}, nil
		}

		if params, err := m.findHandlerParams(r, child); err == nil {
			return child, params, nil
		}
	}

	return route{}, []httpParam{}, errNoMatch
}

// findHandlerParams
func (m *Middleware) findHandlerParams(r *http.Request, child route) ([]httpParam, error) {
	var params []httpParam

	steps := strings.Split(path.Clean(r.URL.Path), "/")

	if len(steps) != len(child.parts) {
		return nil, errNoMatch
	}

	for idx, part := range child.parts {
		if part.root {
			continue
		}

		if part.dyna {
			params = append(params, httpParam{
				Name:  part.name[1:],
				Value: steps[idx],
			})
			continue
		}

		// reset params; invalid route.
		if steps[idx] != part.name {
			return nil, errNoMatch
		}
	}

	return params, nil
}
