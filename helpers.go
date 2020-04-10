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
func Text(w http.ResponseWriter, r *http.Request, v string) (int, error) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	return w.Write([]byte(v))
}

// JSON responds to a request with arbitrary data in JSON format.
func JSON(w http.ResponseWriter, r *http.Request, v interface{}) error {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	return json.NewEncoder(w).Encode(v)
}

// HTML responds to a request with an arbitrary string as HTML.
func HTML(w http.ResponseWriter, r *http.Request, v string) (int, error) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	return w.Write([]byte(v))
}

// Data responds to a request with an arbitrary slice of bytes.
func Data(w http.ResponseWriter, r *http.Request, v []byte) (int, error) {
	w.Header().Set("Content-Type", "application/octet-stream")
	return w.Write(v)
}

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
