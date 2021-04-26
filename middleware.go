package middleware

import (
	"context"
	"errors"
	"net/http"
	"path"
	"strings"
	"time"
)

// nohost is the default hostname unique identifier.
const nohost = "_"

// errNoMatch represents a simple matching error.
var errNoMatch = errors.New("no matching route")

// Middleware is the base of the library and the entry point for every HTTP
// request. It acts as a modular interface that wraps around http.Handler to
// add additional functionality like custom routes, separated HTTP method
// processors and named parameters.
type Middleware struct {
	// Logger records a history of page requests.
	//
	// The W3C maintains a standard format (the Common Log Format) for web
	// server log files, but other proprietary formats exist. More recent
	// entries are typically appended to the end of the file. Information about
	// the request, including client IP address, request date/time, page
	// requested, HTTP code, bytes served, user agent, and referrer are
	// typically added. This data can be combined into a single file, or
	// separated into distinct logs, such as an access log, error log, or
	// referrer log. However, server logs typically do not collect
	// user-specific information.
	//
	// A Logger represents an active logging object that generates lines of
	// output to an io.Writer. Each logging operation makes a single call to
	// the Writer's Write method. A Logger can be used simultaneously from
	// multiple goroutines; it guarantees to serialize access to the Writer.
	//
	// Ref: https://en.wikipedia.org/wiki/Server_log
	Logger Logger

	// NotFound handles page requests to non-existing endpoints.
	//
	// The HTTP 404, 404 Not Found, 404, 404 Error, Page Not Found, File Not
	// Found, or Server Not Found error message is a Hypertext Transfer
	// Protocol (HTTP) standard response code, in computer network
	// communications, to indicate that the browser was able to communicate
	// with a given server, but the server could not find what was requested.
	// The error may also be used when a server does not wish to disclose
	// whether it has the requested information.
	//
	// The website hosting server will typically generate a "404 Not Found" web
	// page when a user attempts to follow a broken or dead link; hence the 404
	// error is one of the most recognizable errors encountered on the World
	// Wide Web.
	NotFound http.Handler

	// IdleTimeout is the maximum amount of time to wait for the next request
	// when keep-alives are enabled. If IdleTimeout is zero, the value of
	// ReadTimeout is used. If both are zero, there is no timeout.
	IdleTimeout time.Duration

	// ReadTimeout is the maximum duration for reading the entire request,
	// including the body. Because ReadTimeout does not let Handlers make
	// per-request decisions on each request body's acceptable deadline or
	// upload rate, most users will prefer to use ReadHeaderTimeout. It is
	// valid to use them both.
	ReadTimeout time.Duration

	// WriteTimeout is the maximum duration before timing out writes of the
	// response. It is reset whenever a new request's header is read. Like
	// ReadTimeout, it does not let Handlers make decisions on a per-request
	// basis.
	WriteTimeout time.Duration

	// ShutdownTimeout is the maximum duration before cancelling the server
	// shutdown context. This allows the developer to guarantee the termination
	// of the server even if a client is keeping a connection idle.
	ShutdownTimeout time.Duration

	// ReadHeaderTimeout is the amount of time allowed to read request headers.
	// The connection's read deadline is reset after reading the headers and
	// the Handler can decide what is considered too slow for the body. If
	// ReadHeaderTimeout is zero, the value of ReadTimeout is used. If both are
	// zero, there is no timeout.
	ReadHeaderTimeout time.Duration

	chain func(http.Handler) http.Handler

	hosts map[string]*Router

	serverInstance *http.Server

	serverShutdown chan bool
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
// `/dev/stdout`. You can disable this by setting `m.Logger = nil` where "m" is
// an instance of `middleware.New()`. You can also writes the logs to a buffer
// or any other Go logger interface defined as `log.New()`.
func New() *Middleware {
	m := new(Middleware)

	m.Logger = NewBasicLogger() /* basic access logger */
	m.hosts = map[string]*Router{nohost: newRouter()}

	return m
}

// DiscardLogs writes all the logs to `/dev/null`.
func (m *Middleware) DiscardLogs() {
	m.Logger = &EmptyLogger{}
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
	router := m.hosts[nohost]

	// Use the host specific router, if available.
	if hostRouter, ok := m.hosts[r.Host]; ok && hostRouter != nil {
		router = hostRouter
	}

	if router == nil {
		http.Error(w, "Unexpected host "+r.Host, http.StatusInternalServerError)
		return
	}

	start := time.Now()
	writer := response{w, 0, 0}
	m.handleRequest(router, &writer, r)
	dur := time.Since(start)

	m.Logger.Log(AccessLog{
		StartTime:     start,
		Host:          r.Host,
		RemoteAddr:    r.RemoteAddr,
		Method:        r.Method,
		Path:          r.URL.Path,
		Query:         r.URL.Query(),
		Protocol:      r.Proto,
		StatusCode:    writer.Status,
		BytesReceived: r.ContentLength,
		BytesSent:     writer.Length,
		Header:        r.Header,
		Duration:      dur,
	})
}

// handleRequest responds to an HTTP request.
//
// The function selects the HTTP handler by traversing a tree that contains a
// list of all the defined URLs without the dynamic parameters (if any). If the
// defined URL doesn’t contains dynamic parameters, the function executes the
// HTTP handler immediately if the URL path matches the request. If there are
// dynamic parameters, the function checks if the URL contains enough data to
// extract them, if there is not enough data, it responds with "404 Not Found",
// otherwise, it attaches the values for the corresponding parameters to the
// request context, then executes the HTTP handler.
//
// Here is an example of a successful request:
//
//   Defined URL: /foo/bar/:group
//   Request URL: /foo/bar/example
//
// This request returns a "200 OK" and the HTTP handler can then obtain a copy
// of the value for the "group" parameter using `middleware.Param()`. Or simply
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
// All these requests will return "404 Not Found" because none of them matches
// the defined URL. This is because trailing slashes are ignored, so even the
// first attempt (which is similar to what the HTTP handler is expecting) will
// fail as there is not enough data to set the value for the "group" parameter.
func (m *Middleware) handleRequest(router *Router, w http.ResponseWriter, r *http.Request) {
	children, ok := router.nodes[r.Method]

	if !ok {
		// HTTP method not allowed, return "405 Method Not Allowed".
		http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
		return
	}

	if r.URL.Path == "" || r.URL.Path[0] != '/' {
		// URL prefix is invalid, return "400 Bad Request".
		http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}

	var handler http.Handler

	child, params, err := m.findHandler(r, children)

	// send "404 Not Found" if there is no handler.
	if err != nil || child.dispatcher == nil {
		handler = m.notFoundHandler()
	} else {
		handler = child.dispatcher
	}

	if len(params) > 0 {
		// insert request parameters into the request context.
		r = r.WithContext(context.WithValue(r.Context(), paramsKey, params))
	}

	if m.chain != nil {
		// pass request through other middlewares.
		m.chain(handler).ServeHTTP(w, r)
		return
	}

	handler.ServeHTTP(w, r)
}

// notFoundHandler returns a request handler that replies to each request with
// a "404 page not found" message, either using custom code attached to the
// router via Middleware.NotFound or with the default Go HTTP package.
func (m *Middleware) notFoundHandler() http.Handler {
	if m.NotFound != nil {
		// custom 404 http handler.
		return m.NotFound
	}

	// default 404 http handler.
	return http.NotFoundHandler()
}

// findHandler returns a request handler that corresponds to the request URL.
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

// findHandlerParams returns the URL parameters associated to the request path.
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

// Host registers a new Top-Level Domain (TLD), if necessary, and then returns
// a pointer to the associated router, which users can use to register an HTTP
// handler of type GET, POST, PUT, PATCH, DELETE, HEAD or OPTIONS to handle
// requests when req.Host == tld.
func (m *Middleware) Host(tld string) *Router {
	if _, ok := m.hosts[tld]; !ok {
		m.hosts[tld] = newRouter()
	}

	return m.hosts[tld]
}

// Handle registers the handler for the given pattern.
func (m *Middleware) Handle(method string, path string, fn http.HandlerFunc) {
	m.hosts[nohost].Handle(method, path, fn)
}

// GET registers a GET endpoint for the default host.
func (m *Middleware) GET(path string, fn http.HandlerFunc) {
	m.hosts[nohost].GET(path, fn)
}

// POST registers a POST endpoint for the default host.
func (m *Middleware) POST(path string, fn http.HandlerFunc) {
	m.hosts[nohost].POST(path, fn)
}

// PUT registers a PUT endpoint for the default host.
func (m *Middleware) PUT(path string, fn http.HandlerFunc) {
	m.hosts[nohost].PUT(path, fn)
}

// PATCH registers a PATCH endpoint for the default host.
func (m *Middleware) PATCH(path string, fn http.HandlerFunc) {
	m.hosts[nohost].PATCH(path, fn)
}

// DELETE registers a DELETE endpoint for the default host.
func (m *Middleware) DELETE(path string, fn http.HandlerFunc) {
	m.hosts[nohost].DELETE(path, fn)
}

// HEAD registers a HEAD endpoint for the default host.
func (m *Middleware) HEAD(path string, fn http.HandlerFunc) {
	m.hosts[nohost].HEAD(path, fn)
}

// OPTIONS registers an OPTIONS endpoint for the default host.
func (m *Middleware) OPTIONS(path string, fn http.HandlerFunc) {
	m.hosts[nohost].OPTIONS(path, fn)
}

// CONNECT registers a CONNECT endpoint for the default host.
func (m *Middleware) CONNECT(path string, fn http.HandlerFunc) {
	m.hosts[nohost].CONNECT(path, fn)
}

// TRACE registers a TRACE endpoint for the default host.
func (m *Middleware) TRACE(path string, fn http.HandlerFunc) {
	m.hosts[nohost].TRACE(path, fn)
}

// COPY registers a WebDAV COPY endpoint for the default host.
func (m *Middleware) COPY(path string, fn http.HandlerFunc) {
	m.hosts[nohost].COPY(path, fn)
}

// LOCK registers a WebDAV LOCK endpoint for the default host.
func (m *Middleware) LOCK(path string, fn http.HandlerFunc) {
	m.hosts[nohost].LOCK(path, fn)
}

// STATIC registers an endpoint to handle GET and POST requests to static files
// in a folder. The function registers the endpoints against the default host.
// The function returns "404 Not Found" if the file does not exist or if the
// client is trying to execute a directory listing attack.
func (m *Middleware) STATIC(folder string, urlPrefix string) {
	m.hosts[nohost].STATIC(folder, urlPrefix)
}
