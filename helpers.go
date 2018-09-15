package middleware

import (
	"net/http"
	"strings"
)

// remoteAddr returns the IP address of the origin of the request.
func remoteAddr(r *http.Request) string {
	mark := strings.Index(r.RemoteAddr, ":")

	if mark == -1 {
		return r.RemoteAddr
	}

	return r.RemoteAddr[0:mark]
}

// inArray checks if the text is in the list.
func inArray(haystack []string, needle string) bool {
	for _, value := range haystack {
		if value == needle {
			return true
		}
	}

	return false
}
