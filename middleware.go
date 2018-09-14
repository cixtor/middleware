package middleware

import (
	"context"
	"crypto/tls"
	"errors"
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

// contextKey is the key for the parameters in the request Context.
type contextKey string

// paramsKey is the key for the parameters in the request Context.
var paramsKey = contextKey("MiddlewareParameter")

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

// httpParam represents a single parameter in the URL.
type httpParam struct {
	Name  string
	Value string
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

	var err error
	var params []httpParam

	for _, child := range children {
		params, err = parseReqParams(r, child)

		if err != nil {
			continue
		}

		if len(params) == 0 {
			child.dispatcher(w, r)
			return
		}

		child.dispatcher(w, r.WithContext(
			context.WithValue(
				r.Context(),
				paramsKey,
				params,
			)))
		return
	}

	if m.NotFound != nil {
		m.NotFound.ServeHTTP(w, r)
		return
	}

	http.NotFound(w, r)
}

// parseReqParams returns a list of request parameters (which may be empty) or
// an error if the requested URL doesn’t match the URL defined in the Btree.
func parseReqParams(r *http.Request, child *Node) ([]httpParam, error) {
	// The URL matches and there are no dynamic parameters.
	//
	// defined: /lorem/ipsum/dolor
	// request: /lorem/ipsum/dolor
	// execute: http.handler
	if child.path == r.URL.Path && child.params == nil {
		return []httpParam{}, nil
	}

	// URL doesn’t match (no dynamic parameters).
	//
	// defined: /lorem/ipsum
	// request: /lorem/ipsum/dolor
	// execute: continue
	//
	// In the example above, the requested URL apparently matches the one
	// defined before, but there seems to be a dynamic parameter that was not
	// expected. Continue iterating until the dispatcher can find an URL that
	// both matches the path and the list of dynamic parameters.
	if child.params == nil {
		return nil, errors.New("URL doesn’t match (no dynamic parameters)")
	}

	// Defined URL is greater or equal than the requested URL.
	//
	// defined: /lorem/ipsum/:example
	// request: /lorem/ipsum
	// execute: continue
	//
	// In the example above, the defined URL is cut in half to separate the
	// static path from the list of dynamic parameter. This causes both the
	// static URL and the requested URL to match, but the existence of dynamic
	// parameters forces the operation to stop because there is not enough
	// information in the URL to set a value for the parameter.
	lendef := len(child.path)
	lenreq := len(r.URL.Path)
	if lendef >= lenreq {
		return nil, errors.New("defined URL is greater or equal than the requested URL")
	}

	// URL doesn’t match (with dynamic parameters).
	//
	// defined: /lorem/ipsum/:example
	// request: /hello/world/something
	// execute: continue
	//
	// In the example above, the length of the defined URL (after removing
	// the dynamic parameter) matches the length of the requested URL after
	// extracting the information used to set the value for the parameters.
	// However, the two remaining static URLs do not match.
	if child.path != r.URL.Path[0:lendef] {
		return nil, errors.New("URL doesn’t match (with dynamic parameters)")
	}

	// Handle request for static files.
	if child.matchEverything {
		return []httpParam{}, nil
	}

	// Separate dynamic parameters from requested URL.
	params := make([]httpParam, child.numParams)
	rawParams := r.URL.Path[lendef+1 : lenreq]
	values := strings.Split(rawParams, "/")

	if len(values) != child.numParams {
		return []httpParam{}, errors.New("incorrect number of dynamic parameters")
	}

	for idx := 0; idx < child.numParams; idx++ {
		params[idx] = httpParam{
			Name:  child.params[idx],
			Value: values[idx],
		}
	}

	return params, nil
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

	m.handleRequest(&writer, r)

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
func Param(r *http.Request, key string) string {
	params, ok := r.Context().Value(paramsKey).([]httpParam)

	if !ok {
		return ""
	}

	for _, param := range params {
		if param.Name == key {
			return param.Value
		}
	}

	return ""
}
