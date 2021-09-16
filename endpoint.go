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
