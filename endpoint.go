package middleware

import (
	"net/http"
	"path"
	"strings"
)

// sep represents the endpoint folder separator.
var sep string = "/"

// endpoint is a data structure to keep the defined routes, named parameters
// and HTTP handler. Some routes like the document root and static files might
// set another property to force the ServeHTTP method to return immediately for
// every match in the URL no matter if the named parameters do not match.
type endpoint struct {
	// Handler is an HTTP handler to process requests to this endpoint.
	Handler http.Handler
	// Folders is a list of segments in the URL with some properties.
	Folders []folder
	// Levels is the number of valid segments in the URL.
	Levels int
	// HasGlob is True if at least one of the folders has a global match mark.
	HasGlob bool
}

// String returns the canonical version of the URL.
func (e endpoint) String() string {
	n := len(e.Folders)
	arr := make([]string, n)
	for i := 0; i < n; i++ {
		arr[i] = string(e.Folders[i])
	}
	return path.Clean(strings.Join(arr, sep))
}

// parseEndpoint decomposes a URL path into basic components.
func parseEndpoint(str string, fn http.Handler) endpoint {
	end := endpoint{
		Handler: fn,
		Folders: []folder{},
	}

	arr := strings.Split(path.Clean(str), sep)

	for _, segment := range arr {
		end.Folders = append(end.Folders, folder(segment))
		end.Levels++

		if segment == "*" {
			end.HasGlob = true
			break
		}
	}

	return end
}

// Match returns True if the endpoint can handle the specified request.
//
// If the endpoint was defined with one or more dynamic parameters, then the
// function will also return a list with all the parameters matched to their
// corresponding values based on the same request.
func (e endpoint) Match(arr []string) ([]httpParam, bool) {
	reqN := len(arr)
	endN := len(e.Folders)

	if endN > reqN {
		// Endpoint expects more folders than the ones available in the URL.
		//
		// - req: /usr/local/etc/openssl
		// - end: /usr/local/:group/:package/cert.pem
		return nil, false
	}

	if endN < reqN && !e.Folders[endN-1].IsGlob() {
		// Endpoint has less folders than the ones in the requested URL, and
		// the last folder in the list is not marked with an asterisk, so we
		// cannot use the global match algorithm.
		return nil, false
	}

	params := []httpParam{}
	matches := make([]bool, endN)

	for i, section := range e.Folders {
		if !section.IsParam() && !section.IsGlob() && string(section) == arr[i] {
			// Exact same text in this part of the URL.
			//
			//         vvv
			// - req: /usr/local/etc/openssl
			// - end: /usr/local/:group/:package/cert.pem
			//             ^^^^^
			matches[i] = true
			continue
		}

		if section.IsParam() {
			params = append(params, httpParam{
				Name:  section.Name(),
				Value: arr[i],
			})
			matches[i] = true
		}

		if i > 0 && section.IsGlob() {
			// Once we find the first global match mark, we can safely ignore
			// the other segments in the request URL and stop the iterator.
			matches[i] = true
			break
		}
	}

	// TODO: optimize; long URLs create big arrays.
	for i := 0; i < endN; i++ {
		if !matches[i] {
			return nil, false
		}
	}

	return params, true
}
