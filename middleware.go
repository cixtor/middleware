package middleware

import (
	"context"
	"log"
	"net/http"
	"strings"
	"time"
)

// DefaultPort is the TCP port number to attach the server.
const DefaultPort = "8080"

// DefaultHost is the IP address to attach the web server.
const DefaultHost = "0.0.0.0"

// Middleware is the base of the library and the entry point for
// every HTTP request. It acts as a modular interface that wraps
// around http.Handler to add additional functionality like
// custom routes, separated HTTP method processors and named
// parameters.
type Middleware struct {
	Host         string
	Port         string
	Nodes        map[string][]*Node
	NotFound     http.Handler
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
}

// StatusWriter is an interface used by an HTTP handler to
// construct an HTTP response. A ResponseWriter may not be used
// after the Handler.ServeHTTP method has returned. Here it is
// being used to include additional data for the logger such as
// the time spent responding to the HTTP request and the total
// size in bytes of the response.
type StatusWriter struct {
	http.ResponseWriter
	Status int
	Length int
}

// Node is a data structure to keep the defined routes, named
// parameters and HTTP handler. Some routes like the document
// root and static files might set another property to force the
// ServeHTTP method to return immediately for every match in the
// URL no matter if the named parameters do not match.
type Node struct {
	Path            string
	Params          []string
	NumParams       int
	NumSections     int
	Dispatcher      http.HandlerFunc
	MatchEverything bool
}

// New returns a new initialized Middleware.
func New() *Middleware {
	return &Middleware{Nodes: make(map[string][]*Node)}
}

// WriteHeader sends an HTTP response header with status code.
// If WriteHeader is not called explicitly, the first call to
// Write will trigger an implicit WriteHeader(http.StatusOK).
// Thus explicit calls to WriteHeader are mainly used to send
// error codes.
func (w *StatusWriter) WriteHeader(status int) {
	w.Status = status
	w.ResponseWriter.WriteHeader(status)
}

// Write writes the data to the connection as part of an HTTP reply.
//
// If WriteHeader has not yet been called, Write calls
// WriteHeader(http.StatusOK) before writing the data. If the Header
// does not contain a Content-Type line, Write adds a Content-Type set
// to the result of passing the initial 512 bytes of written data to
// DetectContentType.
//
// Depending on the HTTP protocol version and the client, calling
// Write or WriteHeader may prevent future reads on the
// Request.Body. For HTTP/1.x requests, handlers should read any
// needed request body data before writing the response. Once the
// headers have been flushed (due to either an explicit Flusher.Flush
// call or writing enough data to trigger a flush), the request body
// may be unavailable. For HTTP/2 requests, the Go HTTP server permits
// handlers to continue to read the request body while concurrently
// writing the response. However, such behavior may not be supported
// by all HTTP/2 clients. Handlers should read before writing if
// possible to maximize compatibility.
func (w *StatusWriter) Write(b []byte) (int, error) {
	if w.Status == 0 {
		w.Status = 200
	}

	w.Length = len(b)

	return w.ResponseWriter.Write(b)
}

// ListenAndServe listens on the TCP network address srv.Addr
// and then calls server.Serve to handle requests on incoming
// connections. Accepted connections are configured to enable
// TCP keep-alives. If srv.Addr is blank, ":http" is used. The
// method always returns a non-nil error.
func (m *Middleware) ListenAndServe() {
	if m.Host == "" {
		m.Host = DefaultHost
	}

	if m.Port == "" {
		m.Port = DefaultPort
	}

	address := m.Host + ":" + m.Port
	server := &http.Server{
		Addr:         address,
		Handler:      m, /* http.DefaultServeMux */
		ReadTimeout:  m.ReadTimeout * time.Second,
		WriteTimeout: m.WriteTimeout * time.Second,
	}

	log.Println("Running server on", address)
	log.Println("PANIC:", server.ListenAndServe())
}

// Dispatcher responds to an HTTP request.
//
// ServeHTTP should write reply headers and data to the ResponseWriter
// and then return. Returning signals that the request is finished; it
// is not valid to use the ResponseWriter or read from the
// Request.Body after or concurrently with the completion of the
// ServeHTTP call.
//
// Depending on the HTTP client software, HTTP protocol version, and
// any intermediaries between the client and the Go server, it may not
// be possible to read from the Request.Body after writing to the
// ResponseWriter. Cautious handlers should read the Request.Body
// first, and then reply.
//
// Except for reading the body, handlers should not modify the
// provided Request.
//
// If ServeHTTP panics, the server (the caller of ServeHTTP) assumes
// that the effect of the panic was isolated to the active request.
// It recovers the panic, logs a stack trace to the server error log,
// and hangs up the connection.
func (m *Middleware) Dispatcher(w http.ResponseWriter, r *http.Request) {
	children, ok := m.Nodes[r.Method]

	if !ok {
		/* Internal server error if HTTP method is not allowed */
		http.Error(w, http.StatusText(405), http.StatusMethodNotAllowed)
		return
	}

	if r.URL.Path == "" || r.URL.Path[0] != '/' {
		/* Bad request error if URL does not starts with slash */
		http.Error(w, http.StatusText(400), http.StatusBadRequest)
		return
	}

	var ctx = r.Context()
	var parameters []string
	var sections []string

	var lendef int   // Length defined URL
	var lenreq int   // Length requested URL
	var extra string // Dynamic URL parameters

	for _, child := range children {
		/* If URL matches and there are no dynamic parameters */
		if child.Path == r.URL.Path && child.Params == nil {
			child.Dispatcher(w, r)
			return
		}

		/* Continue only if the defined URL contains dynamic params. */
		if child.Params == nil {
			continue
		}

		/**
		 * If the defined URL contains dynamic parameters we need to check if
		 * the requested URL is longer than the defined URL without the dynamic
		 * sections, this means that the requested URL must be longer than the
		 * clean defined URL.
		 *
		 * Defined (Raw):   /lorem/ipsum/dolor/:unique
		 * Defined (Clean): /lorem/ipsum/dolor
		 * Req. URL (Bad):  /lorem/ipsum/dolor
		 * Req. URL (Semi): /lorem/ipsum/dolor/
		 * Req. URL (Good): /lorem/ipsum/dolor/something
		 *
		 * Notice how the good requested URL has more characters than the clean
		 * defined URL, the extra characters will be extracted and converted
		 * into variables to be passed to the handler. The bad requested URL
		 * matches the exact same clean defined URL but has no extra characters,
		 * so variable "unique" will be empty which is non-processable. The semi
		 * good requested URL contains one character more than the clean defined
		 * URL, the extra character is simply a forward slash, which means the
		 * dynamic variable "unique" will be empty but at least it was on purpose.
		 *
		 * The requested URL must contains the same characters than the clean
		 * defined URL, at least from index zero, the rest of the requested URL
		 * can be different. This is to prevent malicious requests with semi
		 * valid URLs with different roots which might translate to handlers
		 * processing unrelated requests.
		 */
		lendef = len(child.Path)
		lenreq = len(r.URL.Path)
		if lendef >= lenreq {
			continue
		}

		/* Skip if root section of requested URL does not matches */
		if child.Path != r.URL.Path[0:lendef] {
			continue
		}

		/* Handle request for static files */
		if child.MatchEverything {
			child.Dispatcher(w, r)
			return
		}

		/* Separate dynamic characters from URL */
		parameters = []string{}
		extra = r.URL.Path[lendef:lenreq]
		sections = strings.Split(extra, "/")

		for _, param := range sections {
			if param != "" {
				parameters = append(parameters, param)
			}
		}

		/* Skip if number of dynamic parameters is different */
		if child.NumParams != len(parameters) {
			continue
		}

		for key, name := range child.Params {
			ctx = context.WithValue(ctx, name, parameters[key])
		}

		child.Dispatcher(w, r.WithContext(ctx))
		return
	}

	if m.NotFound != nil {
		m.NotFound.ServeHTTP(w, r)
	} else {
		http.NotFound(w, r)
	}
}

// ServeHTTP dispatches the request to the handler whose pattern
// most closely matches the request URL. Additional to the
// standard functionality this also logs every direct HTTP
// request into the standard output.
func (m *Middleware) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	var query string
	var start = time.Now()
	var writer = StatusWriter{w, 0, 0}

	if r.URL.RawQuery != "" {
		query = "?" + r.URL.RawQuery
	}

	m.Dispatcher(&writer, r)

	log.Printf("%s %s \"%s %s %s\" %d %d \"%s\" %v",
		r.Host,
		r.RemoteAddr,
		r.Method,
		r.URL.Path+query,
		r.Proto,
		writer.Status,
		writer.Length,
		r.Header.Get("User-Agent"),
		time.Now().Sub(start))
}

// ServeFiles serves files from the given file system root. It
// leverages the main functionality to the built-in FileServer
// method exposed by the http package but pre-evaluates the
// request URL to deny direct access to directories and prevent
// directory listing attacks.
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

// Handle registers a new request handle with the given path and method.
//
// This function is intended for bulk loading and to allow the usage of less
// frequently used, non-standardized or custom methods (e.g. for internal
// communication with a proxy).
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

// STATIC refers to the static assets folder, a place where
// people can store files that change with low frequency like
// images, documents, archives and to some extend CSS and
// JavaScript files too. These files are usually better served
// by a cache system and thanks to the design of this library
// you can put one in the middle of your requests as easy as you
// attach normal HTTP handlers.
func (m *Middleware) STATIC(root string, prefix string) {
	var node Node

	node.Path = prefix
	node.MatchEverything = true
	node.Params = []string{"filepath"}
	node.Dispatcher = m.ServeFiles(root, prefix)

	m.Nodes["GET"] = append(m.Nodes["GET"], &node)
	m.Nodes["POST"] = append(m.Nodes["POST"], &node)
}

// GET requests a representation of the specified resource. Note
// that GET should not be used for operations that cause side-
// effects, such as using it for taking actions in web
// applications. One reason for this is that GET may be used
// arbitrarily by robots or crawlers, which should not need to
// consider the side effects that a request should cause.
func (m *Middleware) GET(path string, handle http.HandlerFunc) {
	m.Handle("GET", path, handle)
}

// POST submits data to be processed (e.g., from an HTML form)
// to the identified resource. The data is included in the body
// of the request. This may result in the creation of a new
// resource or the updates of existing resources or both.
//
// Authors of services which use the HTTP protocol SHOULD NOT
// use GET based forms for the submission of sensitive data,
// because this will cause this data to be encoded in the
// Request-URI. Many existing servers, proxies, and user agents
// will log the request URI in some place where it might be
// visible to third parties. Servers can use POST-based form
// submission instead.
func (m *Middleware) POST(path string, handle http.HandlerFunc) {
	m.Handle("POST", path, handle)
}
