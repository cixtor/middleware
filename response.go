package middleware

import (
	"net/http"
)

// response is an interface used by an HTTP handler to construct an HTTP
// response. ResponseWriter may not be used after the Handler.ServeHTTP method
// has returned. Here it’s being used to include additional data for the logger
// such as the time spent responding to the HTTP request and the total size in
// bytes of the response.
type response struct {
	http.ResponseWriter
	Status int
	Length int
}

// WriteHeader sends an HTTP response header with status code.
//
// If WriteHeader is not called explicitly, the first call to Write will
// trigger an implicit WriteHeader(http.StatusOK). Thus explicit calls to
// WriteHeader are mainly used to send error codes.
func (w *response) WriteHeader(status int) {
	w.Status = status
	w.ResponseWriter.WriteHeader(status)
}

// Write writes the data to the connection as part of an HTTP reply.
//
// If WriteHeader hasn’t been called, Write calls WriteHeader(http.StatusOK)
// before writing the data. If the Header does not contain a Content-Type line,
// Write adds a Content-Type set to the result of passing the initial 512 bytes
// of written data to DetectContentType.
//
// Depending on the HTTP protocol version and the client, calling Write or
// WriteHeader may prevent future reads on the Request.Body. For HTTP/1.x
// requests, handlers should read any needed request body data before writing
// the response. Once the headers have been flushed (due to either an explicit
// Flusher.Flush call or writing enough data to trigger a flush), the request
// body may be unavailable. For HTTP/2 requests, the Go HTTP server permits
// handlers to continue to read the request body while concurrently writing the
// response. However, such behavior may not be supported by all HTTP/2 clients.
// Handlers should read before writing if possible to maximize compatibility.
func (w *response) Write(b []byte) (int, error) {
	if w.Status == 0 {
		w.Status = http.StatusOK
	}

	w.Length = len(b)

	return w.ResponseWriter.Write(b)
}
