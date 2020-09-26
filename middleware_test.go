package middleware_test

import (
	"bufio"
	"bytes"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"testing"

	"github.com/cixtor/middleware"
)

var buffer = bytes.Buffer{}
var devnul = bufio.NewWriter(&buffer)
var logger = log.New(devnul, "", log.LstdFlags)

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

func TestIndex(t *testing.T) {
	go func() {
		router := middleware.New()
		router.Logger = logger
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
		router.Logger = logger
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
		router.Logger = logger
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
		router.Logger = logger
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
		router.Logger = logger
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
		router.Logger = logger
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
		router.Logger = logger
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
		router.Logger = logger
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
		router.Logger = logger
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
		router.Logger = logger
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
		router.Logger = logger
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
		router.Logger = logger
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
		router.Logger = logger
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
		router.Logger = logger
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
		router.Logger = logger
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
		router.Logger = logger
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
		router.Logger = logger
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
		router.Logger = logger
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
		router.Logger = logger
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
		router.Logger = logger
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
		router.Logger = logger
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
		router.Logger = logger
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

func TestMultipleHosts(t *testing.T) {
	go func() {
		router := middleware.New()
		router.Logger = logger
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
