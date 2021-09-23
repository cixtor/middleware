package middleware_test

import (
	"bytes"
	"errors"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"syscall"
	"testing"
	"time"

	"github.com/cixtor/middleware"
)

func curl(t *testing.T, method string, hostname string, target string, expected []byte) {
	var err error
	var out []byte
	var req *http.Request
	var res *http.Response

	if req, err = http.NewRequest(method, target, nil); err != nil {
		t.Fatalf("http.NewRequest %s", err)
		return
	}

	req.Host = hostname

	if res, err = http.DefaultClient.Do(req); err != nil {
		t.Fatalf("http.DefaultClient %s", err)
		return
	}

	defer res.Body.Close()

	if out, err = ioutil.ReadAll(res.Body); err != nil {
		t.Fatalf("ioutil.ReadAll %s", err)
		return
	}

	if !bytes.Equal(out, expected) {
		t.Fatalf("%s %s\nexpected: %q\nreceived: %q", method, target, expected, out)
		return
	}
}

func shouldNotCurl(t *testing.T, method string, hostname string, target string) {
	req, err := http.NewRequest(method, target, nil)

	if err != nil {
		t.Fatalf("http.NewRequest %s", err)
		return
	}

	req.Host = hostname

	if _, err := http.DefaultClient.Do(req); errors.Is(err, syscall.ECONNREFUSED) {
		// Detect "connection refused" error and return as a successful call.
		return
	}

	t.Fatalf("%s %s should have failed", method, target)
}

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
	router := middleware.New()
	router.GET("/a/b/c/*", func(w http.ResponseWriter, r *http.Request) { /* ... */ })
	router.DiscardLogs()

	for n := 0; n < b.N; n++ {
		router.ServeHTTP(w, r)
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
		router := middleware.New()
		router.DiscardLogs()
		router.NotFound = x

		router.Handle(method, endpoint, h)

		router.ServeHTTP(w, r)
	})
}

func TestIndex(t *testing.T) {
	go func() {
		router := middleware.New()
		router.DiscardLogs()
		defer router.Shutdown()
		router.GET("/foobar", func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte("Hello World"))
		})
		_ = router.ListenAndServe(":60302")
	}()

	curl(t, "GET", "localhost", "http://localhost:60302/foobar", []byte("Hello World"))
}

func TestUse(t *testing.T) {
	lorem := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("lorem", "lorem")
			next.ServeHTTP(w, r)
		})
	}

	ipsum := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("ipsum", "ipsum")
			next.ServeHTTP(w, r)
		})
	}

	dolor := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("dolor", "dolor")
			next.ServeHTTP(w, r)
		})
	}

	go func() {
		router := middleware.New()
		router.DiscardLogs()
		router.Use(lorem)
		router.Use(ipsum)
		router.Use(dolor)
		defer router.Shutdown()
		router.GET("/foobar", func(w http.ResponseWriter, r *http.Request) {
			a := w.Header().Get("lorem")
			b := w.Header().Get("ipsum")
			c := w.Header().Get("dolor")
			w.Write([]byte(a + ":" + b + ":" + c))
		})
		_ = router.ListenAndServe(":60333")
	}()

	curl(t, "GET", "localhost", "http://localhost:60333/foobar", []byte("lorem:ipsum:dolor"))
}

func TestUse2(t *testing.T) {
	lorem := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte("<1:lorem>"))
			next.ServeHTTP(w, r)
		})
	}

	ipsum := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte("<2:ipsum>"))
			next.ServeHTTP(w, r)
		})
	}

	dolor := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte("<3:dolor>"))
			next.ServeHTTP(w, r)
		})
	}

	go func() {
		router := middleware.New()
		router.DiscardLogs()
		defer router.Shutdown()
		router.GET("/foobar", func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte("<4:foobar>"))
		})
		router.Use(lorem)
		router.Use(ipsum)
		router.Use(dolor)
		_ = router.ListenAndServe(":60334")
	}()

	curl(t, "GET", "localhost", "http://localhost:60334/foobar", []byte("<1:lorem><2:ipsum><3:dolor><4:foobar>"))
}

func TestPOST(t *testing.T) {
	go func() {
		router := middleware.New()
		router.DiscardLogs()
		defer router.Shutdown()
		router.POST("/foobar", func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte("Hello World POST"))
		})
		_ = router.ListenAndServe(":60303")
	}()

	curl(t, "POST", "localhost", "http://localhost:60303/foobar", []byte("Hello World POST"))
}

func TestNotFound(t *testing.T) {
	go func() {
		router := middleware.New()
		router.DiscardLogs()
		defer router.Shutdown()
		router.GET("/", func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte("Hello World GET"))
		})
		_ = router.ListenAndServe(":60304")
	}()

	curl(t, "GET", "localhost", "http://localhost:60304/notfound", []byte("404 page not found\n"))
}

func TestNotFoundSimilar(t *testing.T) {
	go func() {
		router := middleware.New()
		router.DiscardLogs()
		defer router.Shutdown()
		router.GET("/lorem/ipsum/dolor", func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte("Hello World GET"))
		})
		_ = router.ListenAndServe(":60314")
	}()

	curl(t, "GET", "localhost", "http://localhost:60314/lorem/ipsum/dolores", []byte("404 page not found\n"))
}

func TestNotFoundInvalid(t *testing.T) {
	go func() {
		router := middleware.New()
		router.DiscardLogs()
		router.NotFound = nil
		defer router.Shutdown()
		router.GET("/", func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte("Hello World GET"))
		})
		_ = router.ListenAndServe(":60317")
	}()

	curl(t, "GET", "localhost", "http://localhost:60317/test", []byte("404 page not found\n"))
}

func TestNotFoundCustom(t *testing.T) {
	go func() {
		router := middleware.New()
		router.DiscardLogs()
		router.NotFound = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte("404 page does not exist"))
		})
		defer router.Shutdown()
		router.GET("/", func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte("Hello World GET"))
		})
		_ = router.ListenAndServe(":60318")
	}()

	curl(t, "GET", "localhost", "http://localhost:60318/test", []byte("404 page does not exist"))
}

func TestNotFoundCustomWithInterceptor(t *testing.T) {
	go func() {
		router := middleware.New()
		router.DiscardLogs()
		router.NotFound = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte("404 missing page\n"))
		})
		router.Use(func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path == "/test" {
					w.Write([]byte("hello interceptor\n"))
					// return /* do not return */
				}
				next.ServeHTTP(w, r)
			})
		})
		defer router.Shutdown()
		router.GET("/", func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte("Hello World GET"))
		})
		_ = router.ListenAndServe(":60319")
	}()

	curl(t, "GET", "localhost", "http://localhost:60319/test", []byte("hello interceptor\n404 missing page\n"))
}

func TestDirectoryListing(t *testing.T) {
	go func() {
		router := middleware.New()
		router.DiscardLogs()
		defer router.Shutdown()
		router.STATIC(".", "/assets")
		log.Fatal(router.ListenAndServe(":60305"))
	}()

	inputs := [][]string{
		{"test_0", "/assets", "404 page not found\n"},
		{"test_1", "/assets/", "404 page not found\n"},
		{"test_2", "/assets/.git", "Forbidden\n"},
		{"test_3", "/assets/.git/", "Forbidden\n"},
		{"test_4", "/assets/.git/objects", "Forbidden\n"},
		{"test_5", "/assets/.git/objects/", "Forbidden\n"},
	}

	for _, input := range inputs {
		t.Run(input[0], func(t *testing.T) {
			curl(t, "GET", "localhost", "http://localhost:60305"+input[1], []byte(input[2]))
		})
	}
}

func TestSingleParam(t *testing.T) {
	go func() {
		router := middleware.New()
		router.DiscardLogs()
		defer router.Shutdown()
		router.PUT("/hello/:name", func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte(middleware.Param(r, "name")))
		})
		_ = router.ListenAndServe(":60306")
	}()

	curl(t, "PUT", "localhost", "http://localhost:60306/hello/john", []byte("john"))
}

func TestMultiParam(t *testing.T) {
	go func() {
		router := middleware.New()
		router.DiscardLogs()
		defer router.Shutdown()
		router.PATCH("/:group/:section", func(w http.ResponseWriter, r *http.Request) {
			group := middleware.Param(r, "group")
			section := middleware.Param(r, "section")
			w.Write([]byte("page /" + group + "/" + section))
		})
		_ = router.ListenAndServe(":60307")
	}()

	curl(t, "PATCH", "localhost", "http://localhost:60307/account/info", []byte("page /account/info"))
}

func TestMultiParamPrefix(t *testing.T) {
	go func() {
		router := middleware.New()
		router.DiscardLogs()
		defer router.Shutdown()
		router.DELETE("/foo/:group/:section", func(w http.ResponseWriter, r *http.Request) {
			group := middleware.Param(r, "group")
			section := middleware.Param(r, "section")
			w.Write([]byte("page /foo/" + group + "/" + section))
		})
		_ = router.ListenAndServe(":60308")
	}()

	curl(t, "DELETE", "localhost", "http://localhost:60308/foo/account/info", []byte("page /foo/account/info"))
}

func TestComplexParam(t *testing.T) {
	go func() {
		router := middleware.New()
		router.DiscardLogs()
		defer router.Shutdown()
		router.PUT("/account/:name/info", func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte(middleware.Param(r, "name")))
		})
		_ = router.ListenAndServe(":60312")
	}()

	curl(t, "PUT", "localhost", "http://localhost:60312/account/alice/info", []byte("alice"))
}

func TestServeFiles(t *testing.T) {
	go func() {
		router := middleware.New()
		router.DiscardLogs()
		defer router.Shutdown()
		router.STATIC(".", "/cdn")
		_ = router.ListenAndServe(":60311")
	}()

	data, err := ioutil.ReadFile("LICENSE.md")

	if err != nil {
		t.Fatalf("cannot read LICENSE.md %s", err)
		return
	}

	curl(t, "GET", "localhost", "http://localhost:60311/cdn/LICENSE.md", data)
}

func TestServeFilesFake(t *testing.T) {
	go func() {
		router := middleware.New()
		router.DiscardLogs()
		defer router.Shutdown()
		router.GET("/updates/appcast.xml", func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte("<xml></xml>"))
		})
		_ = router.ListenAndServe(":60335")
	}()

	curl(t, "GET", "localhost", "http://localhost:60335/updates/appcast.xml", []byte("<xml></xml>"))
}

func TestServeFilesFakeScript(t *testing.T) {
	go func() {
		router := middleware.New()
		router.DiscardLogs()
		defer router.Shutdown()
		router.GET("/tag/js/gpt.js", func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte("(function(E){})"))
		})
		_ = router.ListenAndServe(":60336")
	}()

	curl(t, "GET", "localhost", "http://localhost:60336/tag/js/gpt.js", []byte("(function(E){})"))
	curl(t, "GET", "localhost", "http://localhost:60336/tag/js/foo.js", []byte("404 page not found\n"))
}

func TestTrailingSlash(t *testing.T) {
	go func() {
		router := middleware.New()
		router.DiscardLogs()
		defer router.Shutdown()
		router.GET("/hello/world/", func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte("Hello World"))
		})
		_ = router.ListenAndServe(":60313")
	}()

	curl(t, "GET", "localhost", "http://localhost:60313/hello/world/", []byte("Hello World"))
}

func TestTrailingSlashDynamic(t *testing.T) {
	go func() {
		router := middleware.New()
		router.DiscardLogs()
		defer router.Shutdown()
		router.POST("/api/:id/store/", func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte("store"))
		})
		_ = router.ListenAndServe(":60316")
	}()

	curl(t, "POST", "localhost", "http://localhost:60316/api/123/store/", []byte("store"))
}

func TestTrailingSlashDynamicMultiple(t *testing.T) {
	go func() {
		router := middleware.New()
		router.DiscardLogs()
		defer router.Shutdown()
		router.POST("/api/:id/store/", func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte("dynamic"))
		})
		_ = router.ListenAndServe(":60324")
	}()

	curl(t, "POST", "localhost", "http://localhost:60324/api/123/////store/", []byte("dynamic"))
}

func TestMultipleRoutes(t *testing.T) {
	go func() {
		router := middleware.New()
		router.DiscardLogs()
		defer router.Shutdown()
		router.GET("/hello/world/", func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte("Hello World"))
		})
		router.GET("/lorem/ipsum/", func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte("Lorem Ipsum"))
		})
		_ = router.ListenAndServe(":60315")
	}()

	curl(t, "GET", "localhost", "http://localhost:60315/hello/world/", []byte("Hello World"))
	curl(t, "GET", "localhost", "http://localhost:60315/lorem/ipsum/", []byte("Lorem Ipsum"))
}

func TestRouteWithAsterisk(t *testing.T) {
	go func() {
		router := middleware.New()
		router.DiscardLogs()
		defer router.Shutdown()
		router.GET("/home/users/*/ignored/sections", func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte("robot"))
		})
		_ = router.ListenAndServe(":60322")
	}()

	curl(t, "GET", "localhost", "http://localhost:60322/home/users/a/b/root", []byte("robot"))
}

func TestRouteWithExtraSlash(t *testing.T) {
	go func() {
		router := middleware.New()
		router.DiscardLogs()
		defer router.Shutdown()
		router.GET("/hello///////world", func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte("hello"))
		})
		_ = router.ListenAndServe(":60323")
	}()

	curl(t, "GET", "localhost", "http://localhost:60323/hello/world", []byte("hello"))
}

func TestRouteWithExtraSlash2(t *testing.T) {
	go func() {
		router := middleware.New()
		router.DiscardLogs()
		defer router.Shutdown()
		router.GET("///////hello/world", func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte("hello"))
		})
		_ = router.ListenAndServe(":60325")
	}()

	curl(t, "GET", "localhost", "http://localhost:60325/hello/world", []byte("hello"))
}

func TestMultipleDynamic(t *testing.T) {
	go func() {
		router := middleware.New()
		router.DiscardLogs()
		defer router.Shutdown()
		router.GET("/hello/:first/:last/info", func(w http.ResponseWriter, r *http.Request) {
			first := middleware.Param(r, "first")
			last := middleware.Param(r, "last")
			w.Write([]byte("Hello " + first + " " + last))
		})
		_ = router.ListenAndServe(":60332")
	}()

	curl(t, "GET", "localhost", "http://localhost:60332/hello/john/smith/info", []byte("Hello john smith"))
}

func TestMultipleHosts(t *testing.T) {
	go func() {
		router := middleware.New()
		router.DiscardLogs()
		defer router.Shutdown()
		router.Host("foo.test").GET("/hello/:name", func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte("@foo.test:" + middleware.Param(r, "name")))
		})
		router.Host("bar.test").GET("/hello/:name", func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte("@bar.test:" + middleware.Param(r, "name")))
		})
		_ = router.ListenAndServe(":60337")
	}()

	curl(t, "GET", "foo.test", "http://localhost:60337/hello/john", []byte("@foo.test:john"))
	curl(t, "GET", "bar.test", "http://localhost:60337/hello/alice", []byte("@bar.test:alice"))
}

func TestDefaultHost(t *testing.T) {
	go func() {
		router := middleware.New()
		router.DiscardLogs()
		defer router.Shutdown()
		router.GET("/hello/:name", func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte("Hello " + middleware.Param(r, "name")))
		})
		router.Host("foo.test").GET("/world/:name", func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte("World " + middleware.Param(r, "name")))
		})
		_ = router.ListenAndServe(":60338")
	}()

	curl(t, "GET", "localhost", "http://localhost:60338/hello/john", []byte("Hello john"))
	curl(t, "GET", "foo.test", "http://localhost:60338/world/earth", []byte("World earth"))
	curl(t, "GET", "bar.test", "http://localhost:60338/anything", []byte("404 page not found\n"))
}

func TestMethodHandle(t *testing.T) {
	go func() {
		router := middleware.New()
		router.DiscardLogs()
		defer router.Shutdown()
		router.Handle("HELLOWORLD", "/foobar", func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte("Hello World"))
		})
		_ = router.ListenAndServe(":60340")
	}()

	curl(t, "HELLOWORLD", "localhost", "http://localhost:60340/foobar", []byte("Hello World"))
}

func TestMethodCONNECT(t *testing.T) {
	go func() {
		router := middleware.New()
		router.DiscardLogs()
		defer router.Shutdown()
		router.CONNECT("/foobar", func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte("Hello World"))
		})
		_ = router.ListenAndServe(":60341")
	}()

	curl(t, "CONNECT", "localhost", "http://localhost:60341/foobar", []byte("Hello World"))
}

func TestMethodTRACE(t *testing.T) {
	go func() {
		router := middleware.New()
		router.DiscardLogs()
		defer router.Shutdown()
		router.TRACE("/foobar", func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte("Hello World"))
		})
		_ = router.ListenAndServe(":60342")
	}()

	curl(t, "TRACE", "localhost", "http://localhost:60342/foobar", []byte("Hello World"))
}

func TestMethodCOPY(t *testing.T) {
	go func() {
		router := middleware.New()
		router.DiscardLogs()
		defer router.Shutdown()
		router.COPY("/foobar", func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte("Hello World"))
		})
		_ = router.ListenAndServe(":60343")
	}()

	curl(t, "COPY", "localhost", "http://localhost:60343/foobar", []byte("Hello World"))
}

func TestMethodLOCK(t *testing.T) {
	go func() {
		router := middleware.New()
		router.DiscardLogs()
		defer router.Shutdown()
		router.LOCK("/foobar", func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte("Hello World"))
		})
		_ = router.ListenAndServe(":60344")
	}()

	curl(t, "LOCK", "localhost", "http://localhost:60344/foobar", []byte("Hello World"))
}

func TestMethodMKCOL(t *testing.T) {
	go func() {
		router := middleware.New()
		router.DiscardLogs()
		defer router.Shutdown()
		router.MKCOL("/foobar", func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte("Hello World"))
		})
		_ = router.ListenAndServe(":60345")
	}()

	curl(t, "MKCOL", "localhost", "http://localhost:60345/foobar", []byte("Hello World"))
}

func TestMethodMOVE(t *testing.T) {
	go func() {
		router := middleware.New()
		router.DiscardLogs()
		defer router.Shutdown()
		router.MOVE("/foobar", func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte("Hello World"))
		})
		_ = router.ListenAndServe(":60346")
	}()

	curl(t, "MOVE", "localhost", "http://localhost:60346/foobar", []byte("Hello World"))
}

func TestMethodPROPFIND(t *testing.T) {
	go func() {
		router := middleware.New()
		router.DiscardLogs()
		defer router.Shutdown()
		router.PROPFIND("/foobar", func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte("Hello World"))
		})
		_ = router.ListenAndServe(":60347")
	}()

	curl(t, "PROPFIND", "localhost", "http://localhost:60347/foobar", []byte("Hello World"))
}

func TestMethodPROPPATCH(t *testing.T) {
	go func() {
		router := middleware.New()
		router.DiscardLogs()
		defer router.Shutdown()
		router.PROPPATCH("/foobar", func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte("Hello World"))
		})
		_ = router.ListenAndServe(":60348")
	}()

	curl(t, "PROPPATCH", "localhost", "http://localhost:60348/foobar", []byte("Hello World"))
}

func TestMethodUNLOCK(t *testing.T) {
	go func() {
		router := middleware.New()
		router.DiscardLogs()
		defer router.Shutdown()
		router.UNLOCK("/foobar", func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte("Hello World"))
		})
		_ = router.ListenAndServe(":60349")
	}()

	curl(t, "UNLOCK", "localhost", "http://localhost:60349/foobar", []byte("Hello World"))
}

func TestEndpointOrder(t *testing.T) {
	go func() {
		router := middleware.New()
		router.DiscardLogs()
		defer router.Shutdown()
		router.GET("/*", func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte("endpoint #7"))
		})
		router.GET("/help/:group/:question/*", func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte("endpoint #6"))
		})
		router.GET("/usr/local/:group/:package/*", func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte("endpoint #5"))
		})
		router.GET("/user/:userid", func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte("endpoint #4"))
		})
		router.GET("/auth/:sso", func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte("endpoint #3"))
		})
		router.GET("/blog/:name/slug", func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte("endpoint #2"))
		})
		router.GET("/help/:page/:group/comments", func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte("endpoint #1"))
		})
		_ = router.ListenAndServe(":60326")
	}()

	inputs := [][]string{
		{"test_0", "/help/viva/family/comments", "endpoint #1"},
		{"test_1", "/blog/hello-world/slug", "endpoint #2"},
		{"test_2", "/auth/openid", "endpoint #3"},
		{"test_3", "/user/12345", "endpoint #4"},
		{"test_4", "/usr/local/etc/openssl/cert.pem", "endpoint #5"},
		{"test_5", "/help/viva/family/foo/bar", "endpoint #6"},
		{"test_6", "/help/viva/family/foobar", "endpoint #6"},
		{"test_7", "/help/viva/family", "endpoint #7"},
		{"test_8", "/any/thing", "endpoint #7"},
		{"test_9", "/hello/world/how/are/you", "endpoint #7"},
	}

	for _, input := range inputs {
		t.Run(input[0], func(t *testing.T) {
			curl(t, "GET", "localhost", "http://localhost:60326"+input[1], []byte(input[2]))
		})
	}
}

func TestAmbiguousPath(t *testing.T) {
	go func() {
		router := middleware.New()
		router.DiscardLogs()
		defer router.Shutdown()
		router.GET("/:package", func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte("package"))
		})
		router.GET("/:package/-/:archive", func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte("package/archive"))
		})
		_ = router.ListenAndServe(":60350")
	}()

	curl(t, "GET", "localhost", "http://localhost:60350/foobar", []byte("package"))
	curl(t, "GET", "localhost", "http://localhost:60350/foobar/-/foobar.tgz", []byte("package/archive"))
}

func TestAmbiguousPath2(t *testing.T) {
	go func() {
		router := middleware.New()
		router.DiscardLogs()
		defer router.Shutdown()
		router.GET("/:package", func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte("package"))
		})
		router.GET("/:module/:package", func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte("module/package"))
		})
		router.GET("/:package/-/:archive", func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte("package/archive"))
		})
		_ = router.ListenAndServe(":60351")
	}()

	curl(t, "GET", "localhost", "http://localhost:60351/foobar", []byte("package"))
	curl(t, "GET", "localhost", "http://localhost:60351/@babel/core", []byte("module/package"))
	curl(t, "GET", "localhost", "http://localhost:60351/foobar/-/foobar.tgz", []byte("package/archive"))
}

func TestAmbiguousPath3(t *testing.T) {
	go func() {
		router := middleware.New()
		router.DiscardLogs()
		defer router.Shutdown()
		router.GET("/usr/local/:group/:user/*", func(w http.ResponseWriter, r *http.Request) {
			group := middleware.Param(r, "group")
			user := middleware.Param(r, "user")
			w.Write([]byte(group + ":" + user))
		})
		_ = router.ListenAndServe(":60327")
	}()

	inputs := [][]string{
		{"should not exist 1", "/usr/local", "404 page not found\n"},
		{"should not exist 2", "/usr/local/etc", "404 page not found\n"},
		{"should not exist 3", "/usr/local/etc/openssl", "404 page not found\n"},
		{"should exist", "/usr/local/etc/openssl/cert.pem", "etc:openssl"},
	}

	for _, input := range inputs {
		t.Run(input[0], func(t *testing.T) {
			curl(t, "GET", "localhost", "http://localhost:60327"+input[1], []byte(input[2]))
		})
	}
}

type telemetry struct {
	called bool
	latest middleware.AccessLog
}

func (t telemetry) ListeningOn(addr string) {}

func (t telemetry) Shutdown(err error) {}

func (t *telemetry) Log(data middleware.AccessLog) {
	t.called = true
	t.latest = data
}

func TestResponseCallback(t *testing.T) {
	tracer := &telemetry{}

	go func() {
		router := middleware.New()
		router.Logger = tracer
		defer router.Shutdown()
		router.GET("/", func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte("Hello World"))
		})
		_ = router.ListenAndServe(":60339")
	}()

	curl(t, "GET", "localhost", "http://localhost:60339/?hello=world&foo=bar", []byte("Hello World"))

	if !tracer.called {
		t.Fatal("http tracer was not called")
	}

	if tracer.latest.Host != "localhost" {
		t.Fatalf("unexpected value for Host: %s", tracer.latest.Host)
	}

	if tracer.latest.RemoteUser != "" {
		t.Fatalf("unexpected value for RemoteUser: %s", tracer.latest.RemoteUser)
	}

	if tracer.latest.Method != "GET" {
		t.Fatalf("unexpected value for Method: %s", tracer.latest.Method)
	}

	if tracer.latest.Path != "/" {
		t.Fatalf("unexpected value for Path: %s", tracer.latest.Path)
	}

	if params := tracer.latest.Query.Encode(); params != "foo=bar&hello=world" {
		t.Fatalf("unexpected value for Query: %s", params)
	}

	if tracer.latest.Protocol != "HTTP/1.1" {
		t.Fatalf("unexpected value for Protocol: %s", tracer.latest.Protocol)
	}

	if tracer.latest.StatusCode != 200 {
		t.Fatalf("unexpected value for StatusCode: %d", tracer.latest.StatusCode)
	}

	if tracer.latest.BytesReceived != 0 {
		t.Fatalf("unexpected value for BytesReceived: %d", tracer.latest.BytesReceived)
	}

	if tracer.latest.BytesSent != 11 {
		t.Fatalf("unexpected value for BytesSent: %d", tracer.latest.BytesSent)
	}

	if ua := tracer.latest.Header.Get("User-Agent"); ua != "Go-http-client/1.1" {
		t.Fatalf("unexpected value for BytesSent: %s", ua)
	}

	if tracer.latest.Duration <= 0 {
		t.Fatalf("unexpected value for Duration: %v", tracer.latest.Duration)
	}
}

func TestShutdown(t *testing.T) {
	router := middleware.New()
	router.DiscardLogs()
	router.GET("/s", func(w http.ResponseWriter, r *http.Request) { w.Write([]byte("xyz")) })

	go router.ListenAndServe(":60309")

	curl(t, "GET", "localhost", "http://localhost:60309/s", []byte("xyz"))
	router.Shutdown()
	shouldNotCurl(t, "GET", "localhost", "http://localhost:60309/s")
}

type CustomSignal int

func (CustomSignal) Signal() {}

func (CustomSignal) String() string { return "custom signal" }

func TestShutdownWithChannel(t *testing.T) {
	done := false
	quit := make(chan os.Signal, 1)
	next := make(chan bool, 1)

	router := middleware.New()
	router.DiscardLogs()
	router.GET("/swc", func(w http.ResponseWriter, r *http.Request) { w.Write([]byte("abc")) })

	go func() {
		<-quit
		router.Shutdown()
		next <- true
		done = true
	}()

	go router.ListenAndServe(":60310")

	curl(t, "GET", "localhost", "http://localhost:60310/swc", []byte("abc"))
	quit <- CustomSignal(60310) // Call middleware.Shutdown to stop the server.

	<-next // Wait for middleware.Shutdown to finish.
	shouldNotCurl(t, "GET", "localhost", "http://localhost:60310/swc")

	if !done {
		t.Fatal("goroutine with middleware.Shutdown did not run correctly")
	}
}

func TestShutdownAddon(t *testing.T) {
	done := false

	router := middleware.New()
	router.DiscardLogs()
	router.OnShutdown = func() { done = true }
	router.GET("/s", func(w http.ResponseWriter, r *http.Request) { w.Write([]byte("XD")) })

	go router.ListenAndServe(":60320")

	curl(t, "GET", "localhost", "http://localhost:60320/s", []byte("XD"))
	router.Shutdown()
	shouldNotCurl(t, "GET", "localhost", "http://localhost:60320/s")

	if !done {
		t.Fatal("middleware.OnShutdown function did not run")
	}
}

var sampleAccessLog = middleware.AccessLog{
	StartTime:     time.Date(2019, time.December, 10, 13, 55, 36, 0, time.UTC),
	Host:          "localhost",
	RemoteAddr:    "127.0.0.1",
	RemoteUser:    "Identity",
	Method:        "POST",
	Path:          "/server-status",
	Query:         url.Values{},
	Protocol:      "HTTP/1.0",
	StatusCode:    200,
	BytesReceived: 0,
	BytesSent:     2326,
	Header: http.Header{
		"Referer":    {"http://www.example.com/"},
		"User-Agent": {"Mozilla/5.0 (KHTML, like Gecko) Version/78.0.3904.108"},
	},
	Duration: time.Millisecond * 5420,
}

func TestLoggerString(t *testing.T) {
	expected := `localhost 127.0.0.1 "POST /server-status HTTP/1.0" 200 2326 "Mozilla/5.0 (KHTML, like Gecko) Version/78.0.3904.108" 5.42s`

	if str := sampleAccessLog.String(); str != expected {
		t.Fatalf("incorrect access log format:\n- %s\n+ %s", expected, str)
	}
}

func TestLoggerCommonLog(t *testing.T) {
	expected := `127.0.0.1 - - [10/12/2019:13:55:36 +00:00] "POST /server-status HTTP/1.0" 200 2326`

	if str := sampleAccessLog.CommonLog(); str != expected {
		t.Fatalf("incorrect common log format:\n- %s\n+ %s", expected, str)
	}
}

func TestLoggerCombinedLog(t *testing.T) {
	expected := `127.0.0.1 - - [10/12/2019:13:55:36 +00:00] "POST /server-status HTTP/1.0" 200 2326 "http://www.example.com/" "Mozilla/5.0 (KHTML, like Gecko) Version/78.0.3904.108"`

	if str := sampleAccessLog.CombinedLog(); str != expected {
		t.Fatalf("incorrect combined log format:\n- %s\n+ %s", expected, str)
	}
}

func TestLoggerCombinedLogWithHyphens(t *testing.T) {
	localAccessLog := sampleAccessLog
	localAccessLog.Header.Set("Referer", "")
	localAccessLog.Header.Set("User-Agent", "")

	expected := `127.0.0.1 - - [10/12/2019:13:55:36 +00:00] "POST /server-status HTTP/1.0" 200 2326 "-" "-"`

	str := sampleAccessLog.CombinedLog()

	if str != expected {
		t.Fatalf("incorrect combined log format:\n- %s\n+ %s", expected, str)
	}
}

type LoggerAndNewLines struct {
	metadata middleware.AccessLog
}

func (l LoggerAndNewLines) ListeningOn(addr string) {}

func (l LoggerAndNewLines) Shutdown(err error) {}

func (l *LoggerAndNewLines) Log(data middleware.AccessLog) {
	l.metadata = data
}

func TestLoggerAndNewLines(t *testing.T) {
	logger := &LoggerAndNewLines{}

	go func() {
		router := middleware.New()
		router.Logger = logger
		defer router.Shutdown()
		router.ListenAndServe(":60321")
	}()

	curl(t, "GET", "localhost", "http://localhost:60321/foo%0abar", []byte("Method Not Allowed\n"))

	expected := `"GET /foo\nbar HTTP/1.1"`

	if str := logger.metadata.Request(); str != expected {
		t.Fatalf("incorrect request section in access log:\n- %s\n+ %s", expected, str)
	}
}
