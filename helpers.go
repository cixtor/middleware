package middleware

import (
	"encoding/json"
	"net/http"
	"strings"
)

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

// Text responds to a request with a string in plain text.
func Text(w http.ResponseWriter, r *http.Request, v string) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")

	if _, err := w.Write([]byte(v)); err != nil {
		w.Write([]byte(err.Error()))
		return
	}
}

// JSON responds to a request with arbitrary data in JSON format.
func JSON(w http.ResponseWriter, r *http.Request, v interface{}) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")

	if err := json.NewEncoder(w).Encode(v); err != nil {
		w.Write([]byte(err.Error()))
		return
	}
}

// HTML responds to a request with an arbitrary string as HTML.
func HTML(w http.ResponseWriter, r *http.Request, v string) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	if _, err := w.Write([]byte(v)); err != nil {
		w.Write([]byte(err.Error()))
		return
	}
}

// func Write()

// remoteAddr returns the IP address of the origin of the request.
//
// When the IP address contains the client port, the function cleans it:
//
//   [::1]:64673 -> [::1]
//   127.0.0.1:64673 -> 127.0.0.1
//   185.153.179.15:64673 -> 185.153.179.15
//   2607:f8b0:400a:808::200e:64673 -> 2607:f8b0:400a:808::200e
func remoteAddr(r *http.Request) string {
	mark := strings.LastIndex(r.RemoteAddr, ":")

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
