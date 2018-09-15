package middleware

import (
	"context"
	"errors"
	"net/http"
	"strings"
)

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
		inArray(m.deniedAddresses, remoteAddr(r)) {
		/* Deny access if the IP address was blacklisted */
		http.Error(w, http.StatusText(403), http.StatusForbidden)
		return
	}

	if m.restrictionType == "DenyAccessExcept" &&
		!inArray(m.allowedAddresses, remoteAddr(r)) {
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
func parseReqParams(r *http.Request, child *route) ([]httpParam, error) {
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
	if child.isStaticHandler {
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
