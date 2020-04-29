package middleware

import (
	"context"
	"errors"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path"
	"strings"
	"time"
)

// defaultShutdownTimeout is the maximum time before server halt.
const defaultShutdownTimeout = 5 * time.Second

// errNoMatch represents a simple matching error.
var errNoMatch = errors.New("no matching route")

// Middleware is the base of the library and the entry point for every HTTP
// request. It acts as a modular interface that wraps around http.Handler to
// add additional functionality like custom routes, separated HTTP method
// processors and named parameters.
type Middleware struct {
	// Hostname (archaically nodename) is a label that is assigned to a device
	// connected to a computer network and that is used to identify the device
	// in various forms of electronic communication, such as the World Wide
	// Web. Hostnames may be simple names consisting of a single word or
	// phrase, or they may be structured. Each hostname usually has at least
	// one numeric network address associated with it for routing packets for
	// performance and other reasons.
	//
	// Internet hostnames may have appended the name of a Domain Name System
	// (DNS) domain, separated from the host-specific label by a period
	// ("dot"). In the latter form, a hostname is also called a domain name. If
	// the domain name is completely specified, including a top-level domain of
	// the Internet, then the hostname is said to be a fully qualified domain
	// name (FQDN). Hostnames that include DNS domains are often stored in the
	// Domain Name System together with the IP addresses of the host they
	// represent for the purpose of mapping the hostname to an address, or the
	// reverse process.
	//
	// Hostnames are composed of a sequence of labels concatenated with dots.
	// For example, "en.example.org" is a hostname. Each label must be from 1
	// to 63 characters long. The entire hostname, including the delimiting
	// dots, has a maximum of 253 ASCII characters.
	//
	// Ref: https://en.wikipedia.org/wiki/Hostname
	Host string

	// Port is a communication endpoint. At the software level, within an
	// operating system, a port is a logical construct that identifies a
	// specific process or a type of network service. A port is identified for
	// each transport protocol and address combination by a 16-bit unsigned
	// number, known as the port number.
	//
	// A port number is a 16-bit unsigned integer, thus ranging from 0 to 65535.
	//
	// For TCP, port number 0 is reserved and cannot be used, while for UDP,
	// the source port is optional and a value of zero means no port.
	//
	// A port number is always associated with an IP address of a host and the
	// type of transport protocol used for communication. It completes the
	// destination or origination network address of a message. Specific port
	// numbers are reserved to identify specific services so that an arriving
	// packet can be easily forwarded to a running application. For this
	// purpose, port numbers lower than 1024 identify the historically most
	// commonly used services and are called the well-known port numbers.
	// Higher-numbered ports are available for general use by applications and
	// are known as ephemeral ports.
	//
	// Ref: https://en.wikipedia.org/wiki/Port_%28computer_networking%29
	Port uint16

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
	Logger *log.Logger

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

	ShutdownTimeout time.Duration

	ReadHeaderTimeout time.Duration

	chain func(http.Handler) http.Handler

	nodes map[string][]route

	serverInstance *http.Server

	serverShutdown chan bool
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

// DiscardLogs writes all the logs to `/dev/null`.
func (m *Middleware) DiscardLogs() {
	m.Logger.SetOutput(ioutil.Discard)
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

	if m.chain != nil {
		// pass request through other middlewares.
		m.chain(child.dispatcher).ServeHTTP(w, r)
		return
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
