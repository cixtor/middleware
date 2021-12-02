package middleware

import (
	"net/http"
	"os"
	"sort"
)

// router is an HTTP routing machine. The default host automatically creates a
// router and all the top-level endpoints are automatically associated to this
// pointer. If the user wants to serve HTTP requests for two different hosts in
// the same web server, they can register the new host to automatically create
// a new routing machine.
type router struct {
	// nodes is a key:value structure where the key represents HTTP methods and
	// the value is a list of endpoints registered to handle HTTP requests at
	// runtime.
	nodes map[string]*privTrie
	// sorted is True if the nodes map is already sorted. The list
	sorted bool
}

// newRouter creates a new instance of the routing machine.
func newRouter() *router {
	return &router{
		nodes: map[string]*privTrie{},
	}
}

// Sort organizes the list of router nodes in such a way that endpoints that
// contain at least one global match mark are evaluated last, and endpoints
// with the most specific path are placed first. This way we avoid conflicts.
func (r *router) Sort() {
	if r.sorted {
		return
	}

	for _, ends := range r.nodes {
		sort.Sort(byEndpoint(ends))
	}

	r.sorted = true
}

// register registers a new request register with the given path and method.
//
// This function is intended for bulk loading and to allow the usage of less
// frequently used, non-standardized or custom methods (e.g. for internal
// communication with a proxy).
func (r *router) register(method string, endpoint string, fn http.Handler) {
	if _, ok := r.nodes[method]; !ok {
		r.nodes[method] = newPrivTrie()
	}
	r.nodes[method].Insert(endpoint, fn)
}

// Handle registers the handler for the given pattern.
func (r *router) Handle(method string, endpoint string, fn http.HandlerFunc) {
	r.register(method, endpoint, fn)
}

// GET requests a representation of the specified resource.
//
// Note that GET should not be used for operations that cause side-effects,
// such as using it for taking actions in web applications. One reason for this
// is that GET may be used arbitrarily by robots or crawlers, which should not
// need to consider the side effects that a request should cause.
func (r *router) GET(endpoint string, fn http.HandlerFunc) {
	r.register(http.MethodGet, endpoint, fn)
}

// POST submits data to be processed to the identified resource.
//
// The data is included in the body of the request. This may result in the
// creation of a new resource or the updates of existing resources or both.
//
// Authors of services which use the HTTP protocol SHOULD NOT use GET based
// forms for the submission of sensitive data, because this will cause this
// data to be encoded in the Request-URI. Many existing servers, proxies, and
// user agents will log the request URI in some place where it might be visible
// to third parties. Servers can use POST-based form submission instead.
func (r *router) POST(endpoint string, fn http.HandlerFunc) {
	r.register(http.MethodPost, endpoint, fn)
}

// PUT is a shortcut for middleware.handle("PUT", endpoint, handle).
func (r *router) PUT(endpoint string, fn http.HandlerFunc) {
	r.register(http.MethodPut, endpoint, fn)
}

// PATCH is a shortcut for middleware.handle("PATCH", endpoint, handle).
func (r *router) PATCH(endpoint string, fn http.HandlerFunc) {
	r.register(http.MethodPatch, endpoint, fn)
}

// DELETE is a shortcut for middleware.handle("DELETE", endpoint, handle).
func (r *router) DELETE(endpoint string, fn http.HandlerFunc) {
	r.register(http.MethodDelete, endpoint, fn)
}

// HEAD is a shortcut for middleware.handle("HEAD", endpoint, handle).
func (r *router) HEAD(endpoint string, fn http.HandlerFunc) {
	r.register(http.MethodHead, endpoint, fn)
}

// OPTIONS is a shortcut for middleware.handle("OPTIONS", endpoint, handle).
func (r *router) OPTIONS(endpoint string, fn http.HandlerFunc) {
	r.register(http.MethodOptions, endpoint, fn)
}

// CONNECT is a shortcut for middleware.handle("CONNECT", endpoint, handle).
func (r *router) CONNECT(endpoint string, fn http.HandlerFunc) {
	r.register(http.MethodConnect, endpoint, fn)
}

// TRACE is a shortcut for middleware.handle("TRACE", endpoint, handle).
func (r *router) TRACE(endpoint string, fn http.HandlerFunc) {
	r.register(http.MethodTrace, endpoint, fn)
}

// COPY is a shortcut for middleware.handle("WebDAV.COPY", endpoint, handle).
func (r *router) COPY(endpoint string, fn http.HandlerFunc) {
	r.register("COPY", endpoint, fn)
}

// LOCK is a shortcut for middleware.handle("WebDAV.LOCK", endpoint, handle).
func (r *router) LOCK(endpoint string, fn http.HandlerFunc) {
	r.register("LOCK", endpoint, fn)
}

// MKCOL is a shortcut for middleware.handle("WebDAV.MKCOL", endpoint, handle).
func (r *router) MKCOL(endpoint string, fn http.HandlerFunc) {
	r.register("MKCOL", endpoint, fn)
}

// MOVE is a shortcut for middleware.handle("WebDAV.MOVE", endpoint, handle).
func (r *router) MOVE(endpoint string, fn http.HandlerFunc) {
	r.register("MOVE", endpoint, fn)
}

// PROPFIND is a shortcut for middleware.handle("WebDAV.PROPFIND", endpoint, handle).
func (r *router) PROPFIND(endpoint string, fn http.HandlerFunc) {
	r.register("PROPFIND", endpoint, fn)
}

// PROPPATCH is a shortcut for middleware.handle("WebDAV.PROPPATCH", endpoint, handle).
func (r *router) PROPPATCH(endpoint string, fn http.HandlerFunc) {
	r.register("PROPPATCH", endpoint, fn)
}

// UNLOCK is a shortcut for middleware.handle("WebDAV.UNLOCK", endpoint, handle).
func (r *router) UNLOCK(endpoint string, fn http.HandlerFunc) {
	r.register("UNLOCK", endpoint, fn)
}

// STATIC refers to the static assets folder, a place where people can store
// files that change with low frequency like images, documents, archives and
// to some extend CSS and JavaScript files too. These files are usually better
// served by a cache system and thanks to the design of this library you can
// put one in the middle of your requests as easy as you attach normal HTTP
// handlers.
func (r *router) STATIC(folder string, urlPrefix string) {
	fn := r.serveFiles(folder, urlPrefix)

	r.HEAD(urlPrefix+"/*", fn)
	r.GET(urlPrefix+"/*", fn)
	r.POST(urlPrefix+"/*", fn)
}

// serveFiles serves files from the root of the given file system.
func (r *router) serveFiles(root string, prefix string) http.HandlerFunc {
	fs := http.FileServer(http.Dir(root))
	handler := http.StripPrefix(prefix, fs)

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fifo, err := os.Stat(root + r.URL.Path[len(prefix):])

		if err != nil {
			// requested resource does not exists; return 404 Not Found
			http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
			return
		}

		if fifo.IsDir() {
			// requested resource is a directory; return 403 Forbidden
			http.Error(w, http.StatusText(http.StatusForbidden), http.StatusForbidden)
			return
		}

		handler.ServeHTTP(w, r)
	})
}
