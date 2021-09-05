package middleware_test

import (
	"bytes"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"syscall"
	"testing"

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

func TestIndex(t *testing.T) {
	go func() {
		router := middleware.New()
		router.DiscardLogs()
		defer router.Shutdown()
		router.GET("/foobar", func(w http.ResponseWriter, r *http.Request) {
			fmt.Fprintf(w, "Hello World")
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
			fmt.Fprintf(
				w,
				"%s:%s:%s",
				w.Header().Get("lorem"),
				w.Header().Get("ipsum"),
				w.Header().Get("dolor"),
			)
		})
		_ = router.ListenAndServe(":60333")
	}()

	curl(t, "GET", "localhost", "http://localhost:60333/foobar", []byte("lorem:ipsum:dolor"))
}

func TestUse2(t *testing.T) {
	lorem := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			fmt.Fprintf(w, "<%s>", "1:lorem")
			next.ServeHTTP(w, r)
		})
	}

	ipsum := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			fmt.Fprintf(w, "<%s>", "2:ipsum")
			next.ServeHTTP(w, r)
		})
	}

	dolor := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			fmt.Fprintf(w, "<%s>", "3:dolor")
			next.ServeHTTP(w, r)
		})
	}

	go func() {
		router := middleware.New()
		router.DiscardLogs()
		defer router.Shutdown()
		router.GET("/foobar", func(w http.ResponseWriter, r *http.Request) {
			fmt.Fprintf(w, "<4:foobar>")
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
			fmt.Fprintf(w, "Hello World POST")
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
			fmt.Fprintf(w, "Hello World GET")
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
			fmt.Fprintf(w, "Hello World GET")
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
			fmt.Fprintf(w, "Hello World GET")
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
			fmt.Fprintf(w, "404 page does not exist\n")
		})
		defer router.Shutdown()
		router.GET("/", func(w http.ResponseWriter, r *http.Request) {
			fmt.Fprintf(w, "Hello World GET")
		})
		_ = router.ListenAndServe(":60318")
	}()

	curl(t, "GET", "localhost", "http://localhost:60318/test", []byte("404 page does not exist\n"))
}

func TestNotFoundCustomWithInterceptor(t *testing.T) {
	go func() {
		router := middleware.New()
		router.DiscardLogs()
		router.NotFound = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			fmt.Fprintf(w, "404 missing page\n")
		})
		router.Use(func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path == "/test" {
					fmt.Fprintf(w, "hello interceptor\n")
					// return /* do not return */
				}
				next.ServeHTTP(w, r)
			})
		})
		defer router.Shutdown()
		router.GET("/", func(w http.ResponseWriter, r *http.Request) {
			fmt.Fprintf(w, "Hello World GET")
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

	curl(t, "GET", "localhost", "http://localhost:60305/assets", []byte("Forbidden\n"))
	curl(t, "GET", "localhost", "http://localhost:60305/assets/", []byte("Forbidden\n"))
	curl(t, "GET", "localhost", "http://localhost:60305/assets/.git", []byte("Forbidden\n"))
	curl(t, "GET", "localhost", "http://localhost:60305/assets/.git/", []byte("Forbidden\n"))
	curl(t, "GET", "localhost", "http://localhost:60305/assets/.git/objects", []byte("Forbidden\n"))
	curl(t, "GET", "localhost", "http://localhost:60305/assets/.git/objects/", []byte("Forbidden\n"))
}

func TestSingleParam(t *testing.T) {
	go func() {
		router := middleware.New()
		router.DiscardLogs()
		defer router.Shutdown()
		router.PUT("/hello/:name", func(w http.ResponseWriter, r *http.Request) {
			fmt.Fprintf(w, "Hello %s", middleware.Param(r, "name"))
		})
		_ = router.ListenAndServe(":60306")
	}()

	curl(t, "PUT", "localhost", "http://localhost:60306/hello/john", []byte("Hello john"))
}

func TestMultiParam(t *testing.T) {
	go func() {
		router := middleware.New()
		router.DiscardLogs()
		defer router.Shutdown()
		router.PATCH("/:group/:section", func(w http.ResponseWriter, r *http.Request) {
			group := middleware.Param(r, "group")
			section := middleware.Param(r, "section")
			fmt.Fprintf(w, "Page /%s/%s", group, section)
		})
		_ = router.ListenAndServe(":60307")
	}()

	curl(t, "PATCH", "localhost", "http://localhost:60307/account/info", []byte("Page /account/info"))
}

func TestMultiParamPrefix(t *testing.T) {
	go func() {
		router := middleware.New()
		router.DiscardLogs()
		defer router.Shutdown()
		router.DELETE("/foo/:group/:section", func(w http.ResponseWriter, r *http.Request) {
			group := middleware.Param(r, "group")
			section := middleware.Param(r, "section")
			fmt.Fprintf(w, "Page /foo/%s/%s", group, section)
		})
		_ = router.ListenAndServe(":60308")
	}()

	curl(t, "DELETE", "localhost", "http://localhost:60308/foo/account/info", []byte("Page /foo/account/info"))
}

func TestComplexParam(t *testing.T) {
	go func() {
		router := middleware.New()
		router.DiscardLogs()
		defer router.Shutdown()
		router.PUT("/account/:name/info", func(w http.ResponseWriter, r *http.Request) {
			fmt.Fprintf(w, "Hello %s", middleware.Param(r, "name"))
		})
		_ = router.ListenAndServe(":60312")
	}()

	curl(t, "PUT", "localhost", "http://localhost:60312/account/john/info", []byte("Hello john"))
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
			fmt.Fprintf(w, "<xml></xml>")
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
			fmt.Fprintf(w, "(function(E){})")
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
			fmt.Fprintf(w, "Hello World")
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
			fmt.Fprintf(w, "store")
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
			fmt.Fprintf(w, "dynamic")
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
			fmt.Fprintf(w, "Hello World")
		})
		router.GET("/lorem/ipsum/", func(w http.ResponseWriter, r *http.Request) {
			fmt.Fprintf(w, "Lorem Ipsum")
		})
		_ = router.ListenAndServe(":60315")
	}()

	curl(t, "GET", "localhost", "http://localhost:60315/hello/world/", []byte("Hello World"))
	curl(t, "GET", "localhost", "http://localhost:60315/lorem/ipsum/", []byte("Lorem Ipsum"))
}

func TestMultipleDynamic(t *testing.T) {
	go func() {
		router := middleware.New()
		router.DiscardLogs()
		defer router.Shutdown()
		router.GET("/hello/:first/:last/info", func(w http.ResponseWriter, r *http.Request) {
			fmt.Fprintf(
				w,
				"Hello %s %s",
				middleware.Param(r, "first"),
				middleware.Param(r, "last"),
			)
		})
		_ = router.ListenAndServe(":60332")
	}()

	curl(t, "GET", "localhost", "http://localhost:60332/hello/john/smith/info", []byte("Hello john smith"))
}

func TestMultipleDynamicACDC(t *testing.T) {
	go func() {
		router := middleware.New()
		router.DiscardLogs()
		defer router.Shutdown()
		router.GET("/bands/:artist/:song", func(w http.ResponseWriter, r *http.Request) {
			fmt.Fprintf(
				w,
				"Band is %s w/ %s",
				middleware.Param(r, "artist"),
				middleware.Param(r, "song"),
			)
		})
		_ = router.ListenAndServe(":60352")
	}()

	curl(t, "GET", "localhost", "http://localhost:60352/bands/AC%2fDC/T.N.T", []byte("Band is AC/DC w/ T.N.T"))
}

func TestMultipleHosts(t *testing.T) {
	go func() {
		router := middleware.New()
		router.DiscardLogs()
		defer router.Shutdown()
		router.Host("foo.test").GET("/hello/:name", func(w http.ResponseWriter, r *http.Request) {
			fmt.Fprintf(w, "Hello from foo.test (%s)", middleware.Param(r, "name"))
		})
		router.Host("bar.test").GET("/hello/:name", func(w http.ResponseWriter, r *http.Request) {
			fmt.Fprintf(w, "Hello from bar.test (%s)", middleware.Param(r, "name"))
		})
		_ = router.ListenAndServe(":60337")
	}()

	curl(t, "GET", "foo.test", "http://localhost:60337/hello/john", []byte("Hello from foo.test (john)"))
	curl(t, "GET", "bar.test", "http://localhost:60337/hello/alice", []byte("Hello from bar.test (alice)"))
}

func TestDefaultHost(t *testing.T) {
	go func() {
		router := middleware.New()
		router.DiscardLogs()
		defer router.Shutdown()
		router.GET("/hello/:name", func(w http.ResponseWriter, r *http.Request) {
			fmt.Fprintf(w, "Hello %s", middleware.Param(r, "name"))
		})
		router.Host("foo.test").GET("/world/:name", func(w http.ResponseWriter, r *http.Request) {
			fmt.Fprintf(w, "World %s", middleware.Param(r, "name"))
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
			fmt.Fprintf(w, "Hello World")
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
			fmt.Fprintf(w, "Hello World")
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
			fmt.Fprintf(w, "Hello World")
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
			fmt.Fprintf(w, "Hello World")
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
			fmt.Fprintf(w, "Hello World")
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
			fmt.Fprintf(w, "Hello World")
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
			fmt.Fprintf(w, "Hello World")
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
			fmt.Fprintf(w, "Hello World")
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
			fmt.Fprintf(w, "Hello World")
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
			fmt.Fprintf(w, "Hello World")
		})
		_ = router.ListenAndServe(":60349")
	}()

	curl(t, "UNLOCK", "localhost", "http://localhost:60349/foobar", []byte("Hello World"))
}

func TestAmbiguousPath(t *testing.T) {
	go func() {
		router := middleware.New()
		router.DiscardLogs()
		defer router.Shutdown()
		router.GET("/:package", func(w http.ResponseWriter, r *http.Request) {
			fmt.Fprintf(w, "package")
		})
		router.GET("/:package/-/:archive", func(w http.ResponseWriter, r *http.Request) {
			fmt.Fprintf(w, "package/archive")
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
			fmt.Fprintf(w, "package")
		})
		router.GET("/:module/:package", func(w http.ResponseWriter, r *http.Request) {
			fmt.Fprintf(w, "module/package")
		})
		router.GET("/:package/-/:archive", func(w http.ResponseWriter, r *http.Request) {
			fmt.Fprintf(w, "package/archive")
		})
		_ = router.ListenAndServe(":60351")
	}()

	curl(t, "GET", "localhost", "http://localhost:60351/foobar", []byte("package"))
	curl(t, "GET", "localhost", "http://localhost:60351/@babel/core", []byte("module/package"))
	curl(t, "GET", "localhost", "http://localhost:60351/foobar/-/foobar.tgz", []byte("package/archive"))
}

type telemetry struct {
	called bool
	latest middleware.AccessLog
}

func (t *telemetry) ListeningOn(addr string) {
}

func (t *telemetry) Shutdown(err error) {
}

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
			fmt.Fprintf(w, "Hello World")
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
	router.GET("/s", func(w http.ResponseWriter, r *http.Request) { fmt.Fprintf(w, "XD") })

	go router.ListenAndServe(":60309")

	curl(t, "GET", "localhost", "http://localhost:60309/s", []byte("XD"))
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
	router.GET("/swc", func(w http.ResponseWriter, r *http.Request) { fmt.Fprintf(w, ";D") })

	go func() {
		<-quit
		router.Shutdown()
		next <- true
		done = true
	}()

	go router.ListenAndServe(":60310")

	curl(t, "GET", "localhost", "http://localhost:60310/swc", []byte(";D"))
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
	router.GET("/s", func(w http.ResponseWriter, r *http.Request) { fmt.Fprintf(w, "XD") })

	go router.ListenAndServe(":60320")

	curl(t, "GET", "localhost", "http://localhost:60320/s", []byte("XD"))
	router.Shutdown()
	shouldNotCurl(t, "GET", "localhost", "http://localhost:60320/s")

	if !done {
		t.Fatal("middleware.OnShutdown function did not run")
	}
}
