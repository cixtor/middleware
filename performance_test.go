package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/cixtor/middleware"
)

type CustomResponseWriter struct {
	body []byte
	code int
	head http.Header
}

func NewCustomResponseWriter() *CustomResponseWriter {
	return &CustomResponseWriter{
		head: http.Header{},
	}
}

func (crw *CustomResponseWriter) Header() http.Header {
	return crw.head
}

func (crw *CustomResponseWriter) Write(b []byte) (int, error) {
	crw.body = b
	return len(b), nil
}

func (crw *CustomResponseWriter) WriteHeader(statusCode int) {
	crw.code = statusCode
}

// BenchmarkServeHTTP checks the performance of the ServeHTTP method.
//
//	go test -bench .
func BenchmarkServeHTTP(b *testing.B) {
	w := NewCustomResponseWriter()
	r := httptest.NewRequest(http.MethodGet, "/a/b/c/d/e/f/g/h/i/j/k/l/m/n/o"+
		"/a/b/c/d/e/f/g/h/i/j/k/l/m/n/o/p/q/r/s/t/u/v/w/x/y/z/a/b/c/d/e/f/g/h"+
		"/i/j/k/l/m/n/o/p/q/r/s/t/u/v/w/x/y/z/a/b/c/d/e/f/g/h/i/j/k/l/m/n/o/p"+
		"/q/r/s/t/u/v/w/x/y/z/p/q/r/s/t/u/v/w/x/y/z/a/b/c/d/e/f/g/h/i/j/k/l/m"+
		"/n/o/p/q/r/s/t/u/v/w/x/y/z/a/b/c/d/e/f/g/h/i/j/k/l/m/n/o/p/q/r/s/t/u"+
		"/v/w/x/y/z", nil)
	srv := middleware.New()
	srv.GET("/a/b/c/*", func(w http.ResponseWriter, r *http.Request) { /* ... */ })
	srv.DiscardLogs()

	for n := 0; n < b.N; n++ {
		srv.ServeHTTP(w, r)
	}
}

// FuzzServeHTTP checks for panics somewhere in the ServeHTTP operations.
//
//	go test -fuzz FuzzServeHTTP -fuzztime 30s
func FuzzServeHTTP(f *testing.F) {
	f.Add("GET", "/")
	f.Add("POST", "/hello/world")
	f.Add("HEAD", "/////hello/world")
	f.Add("PUT", "/server-status")
	f.Add("DELETE", "/a/b/c/d/e/f/g/h/i/j/k/l/m/n/o/p/q/r/s/t/u/v/w/x/y/z")

	w := NewCustomResponseWriter()
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	h := func(w http.ResponseWriter, r *http.Request) { w.Write([]byte("fuzz")) }
	x := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("404 fuzz page"))
	})

	f.Fuzz(func(t *testing.T, method string, endpoint string) {
		srv := middleware.New()
		srv.DiscardLogs()
		srv.NotFound = x

		srv.Handle(method, endpoint, h)

		srv.ServeHTTP(w, r)
	})
}
