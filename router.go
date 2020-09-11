package middleware

import (
	"net/http"
	"os"
	"strings"
)

// Router is an HTTP routing machine. The default host automatically creates a
// router and all the top-level endpoints are automatically associated to this
// pointer. If the user wants to serve HTTP requests for two different hosts in
// the same web server, they can register the new host to automatically create
// a new routing machine.
type Router struct {
	nodes map[string][]route
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
	// dyna is short for "dynamic"; true if `/^:\S+/` false otherwise
	dyna bool
	// root is true if the route part is the first in the list.
	root bool
}

// newRouter creates a new instance of the routing machine.
func newRouter() *Router {
	return &Router{
		nodes: map[string][]route{},
	}
}

// register registers a new request register with the given path and method.
//
// This function is intended for bulk loading and to allow the usage of less
// frequently used, non-standardized or custom methods (e.g. for internal
// communication with a proxy).
func (r *Router) register(method, path string, fn http.HandlerFunc) {
	node := route{path: path, dispatcher: fn}
	parts := strings.Split(path, "/")

	for idx, section := range parts {
		if section == "" && idx == 0 {
			node.parts = append(node.parts, rpart{
				name: "<root>",
				root: true,
			})
			continue
		}

		if section == "" && idx > 0 {
			continue
		}

		if section == "*" {
			node.glob = true
			continue
		}

		if section[0] == ':' {
			node.parts = append(node.parts, rpart{
				name: section,
				dyna: true,
			})
			continue
		}

		node.parts = append(node.parts, rpart{name: section})
	}

	r.nodes[method] = append(r.nodes[method], node)
}

// GET requests a representation of the specified resource.
//
// Note that GET should not be used for operations that cause side-effects,
// such as using it for taking actions in web applications. One reason for this
// is that GET may be used arbitrarily by robots or crawlers, which should not
// need to consider the side effects that a request should cause.
func (r *Router) GET(path string, fn http.HandlerFunc) {
	r.register(http.MethodGet, path, fn)
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
func (r *Router) POST(path string, fn http.HandlerFunc) {
	r.register(http.MethodPost, path, fn)
}

// PUT is a shortcut for middleware.handle("PUT", path, handle).
func (r *Router) PUT(path string, fn http.HandlerFunc) {
	r.register(http.MethodPut, path, fn)
}

// PATCH is a shortcut for middleware.handle("PATCH", path, handle).
func (r *Router) PATCH(path string, fn http.HandlerFunc) {
	r.register(http.MethodPatch, path, fn)
}

// DELETE is a shortcut for middleware.handle("DELETE", path, handle).
func (r *Router) DELETE(path string, fn http.HandlerFunc) {
	r.register(http.MethodDelete, path, fn)
}

// HEAD is a shortcut for middleware.handle("HEAD", path, handle).
func (r *Router) HEAD(path string, fn http.HandlerFunc) {
	r.register(http.MethodHead, path, fn)
}

// OPTIONS is a shortcut for middleware.handle("OPTIONS", path, handle).
func (r *Router) OPTIONS(path string, fn http.HandlerFunc) {
	r.register(http.MethodOptions, path, fn)
}

// STATIC refers to the static assets folder, a place where people can store
// files that change with low frequency like images, documents, archives and
// to some extend CSS and JavaScript files too. These files are usually better
// served by a cache system and thanks to the design of this library you can
// put one in the middle of your requests as easy as you attach normal HTTP
// handlers.
func (r *Router) STATIC(folder string, urlPrefix string) {
	node := route{
		path:       urlPrefix,
		glob:       true,
		dispatcher: r.ServeFiles(folder, urlPrefix),
	}

	r.nodes[http.MethodGet] = append(r.nodes[http.MethodGet], node)
	r.nodes[http.MethodPost] = append(r.nodes[http.MethodPost], node)
}

// ServeFiles serves files from the given file system root.
//
// A pre-check is executed to prevent directory listing attacks.
func (r *Router) ServeFiles(root string, prefix string) http.HandlerFunc {
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
