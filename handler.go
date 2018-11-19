package middleware

import (
	"context"
	"errors"
	"net/http"
	"strings"
)

// errNoMatch is an error for when nothing matches.
var errNoMatch = errors.New("route doesn’t match")

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
	var incorrect bool
	var params []httpParam

	steps := strings.Split(r.URL.Path, "/")

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

		if steps[idx] != part.name {
			incorrect = true
			break
		}
	}

	if incorrect {
		return nil, errNoMatch
	}

	return params, nil
}
