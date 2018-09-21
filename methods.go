package middleware

import (
	"net/http"
	"strings"
)

// handle registers a new request handle with the given path and method.
//
// This function is intended for bulk loading and to allow the usage of less
// frequently used, non-standardized or custom methods (e.g. for internal
// communication with a proxy).
func (m *Middleware) handle(method, path string, handle http.HandlerFunc) {
	var node route
	var usable []string

	if m.chain != nil {
		node.dispatcher = m.chain(handle).ServeHTTP
	} else {
		node.dispatcher = handle
	}

	parts := strings.Split(path, "/")

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

	node.path = "/" + strings.Join(usable, "/")

	m.nodes[method] = append(m.nodes[method], &node)
}

// GET requests a representation of the specified resource.
//
// Note that GET should not be used for operations that cause side-effects,
// such as using it for taking actions in web applications. One reason for this
// is that GET may be used arbitrarily by robots or crawlers, which should not
// need to consider the side effects that a request should cause.
func (m *Middleware) GET(path string, handle http.HandlerFunc) {
	m.handle("GET", path, handle)
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
func (m *Middleware) POST(path string, handle http.HandlerFunc) {
	m.handle("POST", path, handle)
}

// PUT is a shortcut for middleware.handle("PUT", path, handle).
func (m *Middleware) PUT(path string, handle http.HandlerFunc) {
	m.handle("PUT", path, handle)
}

// PATCH is a shortcut for middleware.handle("PATCH", path, handle).
func (m *Middleware) PATCH(path string, handle http.HandlerFunc) {
	m.handle("PATCH", path, handle)
}

// DELETE is a shortcut for middleware.handle("DELETE", path, handle).
func (m *Middleware) DELETE(path string, handle http.HandlerFunc) {
	m.handle("DELETE", path, handle)
}

// HEAD is a shortcut for middleware.handle("HEAD", path, handle).
func (m *Middleware) HEAD(path string, handle http.HandlerFunc) {
	m.handle("HEAD", path, handle)
}

// OPTIONS is a shortcut for middleware.handle("OPTIONS", path, handle).
func (m *Middleware) OPTIONS(path string, handle http.HandlerFunc) {
	m.handle("OPTIONS", path, handle)
}

// STATIC refers to the static assets folder, a place where people can store
// files that change with low frequency like images, documents, archives and
// to some extend CSS and JavaScript files too. These files are usually better
// served by a cache system and thanks to the design of this library you can
// put one in the middle of your requests as easy as you attach normal HTTP
// handlers.
func (m *Middleware) STATIC(root string, prefix string) {
	var node route

	node.path = prefix
	node.isStaticHandler = true
	node.params = []string{"filepath"}
	node.dispatcher = m.ServeFiles(root, prefix)

	m.nodes["GET"] = append(m.nodes["GET"], &node)
	m.nodes["POST"] = append(m.nodes["POST"], &node)
}
