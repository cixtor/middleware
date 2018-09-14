package middleware

import (
	"context"
	"crypto/tls"
	"log"
	"net/http"
	"os"
	"strings"
	"time"
)

// defaultPort is the TCP port number to attach the server.
const defaultPort = "8080"

// defaultHost is the IP address to attach the web server.
const defaultHost = "0.0.0.0"

// defaultShutdownTimeout is the maximum time before server halt.
const defaultShutdownTimeout = 5 * time.Second

// Middleware is the base of the library and the entry point for
// every HTTP request. It acts as a modular interface that wraps
// around http.Handler to add additional functionality like
// custom routes, separated HTTP method processors and named
// parameters.
type Middleware struct {
	Host              string
	Port              string
	NotFound          http.Handler
	IdleTimeout       time.Duration
	ReadTimeout       time.Duration
	WriteTimeout      time.Duration
	ShutdownTimeout   time.Duration
	ReadHeaderTimeout time.Duration
	nodes             map[string][]*Node
	logger            *log.Logger
	serverInstance    *http.Server
	serverShutdown    chan bool
	allowedAddresses  []string
	deniedAddresses   []string
	restrictionType   string
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
	path            string
	params          []string
	numParams       int
	numSections     int
	dispatcher      http.HandlerFunc
	matchEverything bool
}

// New returns a new initialized Middleware.
func New() *Middleware {
	m := new(Middleware)

	m.nodes = make(map[string][]*Node)
	m.logger = log.New(os.Stdout, "", log.LstdFlags)

	return m
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

// setDefaultSettings sets the default server settings.
func (m *Middleware) setDefaultSettings() {
	if m.Host == "" {
		m.Host = defaultHost
	}

	if m.Port == "" {
		m.Port = defaultPort
	}

	if m.ShutdownTimeout == 0 {
		m.ShutdownTimeout = defaultShutdownTimeout
	}
}

// gracefulServerShutdown shutdowns the server.
func (m *Middleware) gracefulServerShutdown() {
	<-m.serverShutdown /* wait shutdown */

	ctx, cancel := context.WithTimeout(
		context.Background(),
		m.ShutdownTimeout,
	)

	defer cancel()

	if err := m.serverInstance.Shutdown(ctx); err != nil {
		m.logger.Println("sigint;", err)
		return
	}

	m.logger.Println("server shutdown")
}

// startWebServer setups and starts the web server.
func (m *Middleware) startWebServer(f func()) {
	m.setDefaultSettings()

	address := m.Host + ":" + m.Port
	m.serverShutdown = make(chan bool)
	m.serverInstance = &http.Server{
		Addr:              address,
		Handler:           m,
		IdleTimeout:       m.IdleTimeout,
		ReadTimeout:       m.ReadTimeout,
		WriteTimeout:      m.WriteTimeout,
		ReadHeaderTimeout: m.ReadHeaderTimeout,
	}

	go func() {
		m.logger.Println("listening on", address)
		f() /* m.ListenAndServe OR m.ListenAndServeTLS */
	}()

	m.gracefulServerShutdown()
}

// ListenAndServe listens on the TCP network address srv.Addr
// and then calls server.Serve to handle requests on incoming
// connections. Accepted connections are configured to enable
// TCP keep-alives. If srv.Addr is blank, ":http" is used. The
// method always returns a non-nil error.
func (m *Middleware) ListenAndServe() {
	m.startWebServer(func() {
		err := m.serverInstance.ListenAndServe()

		if err != nil {
			m.logger.Fatal(err)
		}
	})
}

// ListenAndServeTLS acts identically to ListenAndServe, except that it
// expects HTTPS connections. Additionally, files containing a certificate and
// matching private key for the server must be provided. If the certificate
// is signed by a certificate authority, the certFile should be the concatenation
// of the server's certificate, any intermediates, and the CA's certificate.
func (m *Middleware) ListenAndServeTLS(certFile string, keyFile string, cfg *tls.Config) {
	m.startWebServer(func() {
		m.serverInstance.TLSConfig = cfg /* custom TLLS config */

		err := m.serverInstance.ListenAndServeTLS(certFile, keyFile)

		if err != nil {
			m.logger.Fatal(err)
		}
	})
}

// dispatcher responds to an HTTP request.
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
func (m *Middleware) dispatcher(w http.ResponseWriter, r *http.Request) {
	children, ok := m.nodes[r.Method]

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

	if m.restrictionType == "AllowAccessExcept" &&
		m.inArray(m.deniedAddresses, m.remoteAddr(r)) {
		/* Deny access if the IP address was blacklisted */
		http.Error(w, http.StatusText(403), http.StatusForbidden)
		return
	}

	if m.restrictionType == "DenyAccessExcept" &&
		!m.inArray(m.allowedAddresses, m.remoteAddr(r)) {
		/* Deny access if the IP address is not whitelisted */
		http.Error(w, http.StatusText(403), http.StatusForbidden)
		return
	}

	var ctx = r.Context()
	var params []string

	var lendef int // Length defined URL
	var lenreq int // Length requested URL

	for _, child := range children {
		/* If URL matches and there are no dynamic parameters */
		if child.path == r.URL.Path && child.params == nil {
			child.dispatcher(w, r)
			return
		}

		/* Continue only if the defined URL contains dynamic params. */
		if child.params == nil {
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
		lendef = len(child.path)
		lenreq = len(r.URL.Path)
		if lendef >= lenreq {
			continue
		}

		/* Skip if root section of requested URL does not matches */
		if child.path != r.URL.Path[0:lendef] {
			continue
		}

		/* Handle request for static files */
		if child.matchEverything {
			child.dispatcher(w, r)
			return
		}

		/* Separate dynamic characters from URL */
		params = m.urlParams(r.URL.Path[lendef:lenreq])

		/* Skip if number of dynamic parameters is different */
		if child.numParams != len(params) {
			continue
		}

		for key, name := range child.params {
			ctx = context.WithValue(ctx, name, params[key])
		}

		child.dispatcher(w, r.WithContext(ctx))
		return
	}

	if m.NotFound != nil {
		m.NotFound.ServeHTTP(w, r)
		return
	}

	http.NotFound(w, r)
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

	m.dispatcher(&writer, r)

	m.logger.Printf("%s %s \"%s %s %s\" %d %d \"%s\" %v",
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

// handle registers a new request handle with the given path and method.
//
// This function is intended for bulk loading and to allow the usage of less
// frequently used, non-standardized or custom methods (e.g. for internal
// communication with a proxy).
func (m *Middleware) handle(method, path string, handle http.HandlerFunc) {
	var node Node
	var parts []string
	var usable []string

	node.path = "/"
	node.dispatcher = handle
	parts = strings.Split(path, "/")

	// Separate dynamic parameters from the static URL.
	for _, section := range parts {
		if section == "" {
			continue
		}

		if len(section) > 1 && section[0] == ':' {
			node.params = append(node.params, section[1:])
			node.numSections++
			node.numParams++
			continue
		}

		usable = append(usable, section)
		node.numSections++
	}

	node.path += strings.Join(usable, "/")

	m.nodes[method] = append(m.nodes[method], &node)
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

	node.path = prefix
	node.matchEverything = true
	node.params = []string{"filepath"}
	node.dispatcher = m.ServeFiles(root, prefix)

	m.nodes["GET"] = append(m.nodes["GET"], &node)
	m.nodes["POST"] = append(m.nodes["POST"], &node)
}

// GET requests a representation of the specified resource. Note
// that GET should not be used for operations that cause side-
// effects, such as using it for taking actions in web
// applications. One reason for this is that GET may be used
// arbitrarily by robots or crawlers, which should not need to
// consider the side effects that a request should cause.
func (m *Middleware) GET(path string, handle http.HandlerFunc) {
	m.handle("GET", path, handle)
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
	m.handle("POST", path, handle)
}

// PUT is a shortcut for middleware.handle("PUT", path, handle)
func (m *Middleware) PUT(path string, handle http.HandlerFunc) {
	m.handle("PUT", path, handle)
}

// PATCH is a shortcut for middleware.handle("PATCH", path, handle)
func (m *Middleware) PATCH(path string, handle http.HandlerFunc) {
	m.handle("PATCH", path, handle)
}

// DELETE is a shortcut for middleware.handle("DELETE", path, handle)
func (m *Middleware) DELETE(path string, handle http.HandlerFunc) {
	m.handle("DELETE", path, handle)
}

// HEAD is a shortcut for middleware.handle("HEAD", path, handle)
func (m *Middleware) HEAD(path string, handle http.HandlerFunc) {
	m.handle("HEAD", path, handle)
}

// OPTIONS is a shortcut for middleware.handle("OPTIONS", path, handle)
func (m *Middleware) OPTIONS(path string, handle http.HandlerFunc) {
	m.handle("OPTIONS", path, handle)
}

// urlParams reads, parses and clean a dynamic URL.
func (m *Middleware) urlParams(text string) []string {
	var params []string

	sections := strings.Split(text, "/")

	for _, param := range sections {
		if param != "" {
			params = append(params, param)
		}
	}

	return params
}

// remoteAddr returns the IP address of the origin of the request.
func (m *Middleware) remoteAddr(r *http.Request) string {
	parts := strings.Split(r.RemoteAddr, ":")

	return parts[0]
}

// inArray checks if the text is in the list.
func (m *Middleware) inArray(haystack []string, needle string) bool {
	var exists bool

	for _, value := range haystack {
		if value == needle {
			exists = true
			break
		}
	}

	return exists
}

// Shutdown stops the web server.
func (m *Middleware) Shutdown() {
	m.serverShutdown <- true
}

// AllowAccessExcept returns a "403 Forbidden" if the IP is blacklisted.
func (m *Middleware) AllowAccessExcept(ips []string) {
	m.restrictionType = "AllowAccessExcept"
	m.deniedAddresses = ips
}

// DenyAccessExcept returns a "403 Forbidden" unless the IP is whitelisted.
func (m *Middleware) DenyAccessExcept(ips []string) {
	m.restrictionType = "DenyAccessExcept"
	m.allowedAddresses = ips
}

// Param returns the value for a parameter in the URL.
func Param(r *http.Request, key interface{}) interface{} {
	return r.Context().Value(key)
}
