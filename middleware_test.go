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

func curl(t *testing.T, method string, target string, expected []byte) {
	var err error
	var out []byte
	var req *http.Request
	var res *http.Response

	if req, err = http.NewRequest(method, target, nil); err != nil {
		t.Fatalf("http.NewRequest %s", err)
		return
	}

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
		router.Port = 60302
		defer router.Shutdown()
		router.GET("/foobar", func(w http.ResponseWriter, r *http.Request) {
			fmt.Fprintf(w, "Hello World")
		})
		router.ListenAndServe()
	}()

	curl(t, "GET", "http://localhost:60302/foobar", []byte("Hello World"))
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
		router.Port = 60333
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
		router.ListenAndServe()
	}()

	curl(t, "GET", "http://localhost:60333/foobar", []byte("lorem:ipsum:dolor"))
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
		router.Port = 60334
		defer router.Shutdown()
		router.GET("/foobar", func(w http.ResponseWriter, r *http.Request) {
			fmt.Fprintf(w, "<4:foobar>")
		})
		router.Use(lorem)
		router.Use(ipsum)
		router.Use(dolor)
		router.ListenAndServe()
	}()

	curl(t, "GET", "http://localhost:60334/foobar", []byte("<1:lorem><2:ipsum><3:dolor><4:foobar>"))
}

func TestPOST(t *testing.T) {
	go func() {
		router := middleware.New()
		router.Logger = logger
		router.Port = 60303
		defer router.Shutdown()
		router.POST("/foobar", func(w http.ResponseWriter, r *http.Request) {
			fmt.Fprintf(w, "Hello World POST")
		})
		router.ListenAndServe()
	}()

	curl(t, "POST", "http://localhost:60303/foobar", []byte("Hello World POST"))
}

func TestNotFound(t *testing.T) {
	go func() {
		router := middleware.New()
		router.Logger = logger
		router.Port = 60304
		defer router.Shutdown()
		router.GET("/", func(w http.ResponseWriter, r *http.Request) {
			fmt.Fprintf(w, "Hello World GET")
		})
		router.ListenAndServe()
	}()

	curl(t, "GET", "http://localhost:60304/notfound", []byte("404 page not found\n"))
}

func TestNotFoundSimilar(t *testing.T) {
	go func() {
		router := middleware.New()
		router.Logger = logger
		router.Port = 60314
		defer router.Shutdown()
		router.GET("/lorem/ipsum/dolor", func(w http.ResponseWriter, r *http.Request) {
			fmt.Fprintf(w, "Hello World GET")
		})
		router.ListenAndServe()
	}()

	curl(t, "GET", "http://localhost:60314/dolor/ipsum/lorem", []byte("404 page not found\n"))
}

func TestDirectoryListing(t *testing.T) {
	go func() {
		router := middleware.New()
		router.Logger = logger
		router.Port = 60305
		defer router.Shutdown()
		router.STATIC(".", "/assets")
		router.ListenAndServe()
	}()

	curl(t, "GET", "http://localhost:60305/assets", []byte("Forbidden\n"))
	curl(t, "GET", "http://localhost:60305/assets/", []byte("Forbidden\n"))
	curl(t, "GET", "http://localhost:60305/assets/.git", []byte("Forbidden\n"))
	curl(t, "GET", "http://localhost:60305/assets/.git/", []byte("Forbidden\n"))
	curl(t, "GET", "http://localhost:60305/assets/.git/objects", []byte("Forbidden\n"))
	curl(t, "GET", "http://localhost:60305/assets/.git/objects/", []byte("Forbidden\n"))
}

func TestSingleParam(t *testing.T) {
	go func() {
		router := middleware.New()
		router.Logger = logger
		router.Port = 60306
		defer router.Shutdown()
		router.PUT("/hello/:name", func(w http.ResponseWriter, r *http.Request) {
			fmt.Fprintf(w, "Hello %s", middleware.Param(r, "name"))
		})
		router.ListenAndServe()
	}()

	curl(t, "PUT", "http://localhost:60306/hello/john", []byte("Hello john"))
}

func TestMultiParam(t *testing.T) {
	go func() {
		router := middleware.New()
		router.Logger = logger
		router.Port = 60307
		defer router.Shutdown()
		router.PATCH("/:group/:section", func(w http.ResponseWriter, r *http.Request) {
			group := middleware.Param(r, "group")
			section := middleware.Param(r, "section")
			fmt.Fprintf(w, "Page /%s/%s", group, section)
		})
		router.ListenAndServe()
	}()

	curl(t, "PATCH", "http://localhost:60307/account/info", []byte("Page /account/info"))
}

func TestMultiParamPrefix(t *testing.T) {
	go func() {
		router := middleware.New()
		router.Logger = logger
		router.Port = 60308
		defer router.Shutdown()
		router.DELETE("/foo/:group/:section", func(w http.ResponseWriter, r *http.Request) {
			group := middleware.Param(r, "group")
			section := middleware.Param(r, "section")
			fmt.Fprintf(w, "Page /foo/%s/%s", group, section)
		})
		router.ListenAndServe()
	}()

	curl(t, "DELETE", "http://localhost:60308/foo/account/info", []byte("Page /foo/account/info"))
}

func TestComplexParam(t *testing.T) {
	go func() {
		router := middleware.New()
		router.Logger = logger
		router.Port = 60312
		defer router.Shutdown()
		router.PUT("/account/:name/info", func(w http.ResponseWriter, r *http.Request) {
			fmt.Fprintf(w, "Hello %s", middleware.Param(r, "name"))
		})
		router.ListenAndServe()
	}()

	curl(t, "PUT", "http://localhost:60312/account/john/info", []byte("Hello john"))
}

func TestAllowAccess(t *testing.T) {
	go func() {
		router := middleware.New()
		router.Logger = logger
		router.Port = 60309
		defer router.Shutdown()
		router.AllowAccessExcept([]string{"[::1]"})
		router.OPTIONS("/admin/user/info", func(w http.ResponseWriter, r *http.Request) {
			fmt.Fprintf(w, "Hello World")
		})
		router.ListenAndServe()
	}()

	curl(t, "OPTIONS", "http://localhost:60309/admin/user/info", []byte("Forbidden\n"))
}

func TestDenyAccess(t *testing.T) {
	go func() {
		router := middleware.New()
		router.Logger = logger
		router.Port = 60310
		defer router.Shutdown()
		router.DenyAccessExcept([]string{"82.82.82.82"})
		router.OPTIONS("/admin/user/info", func(w http.ResponseWriter, r *http.Request) {
			fmt.Fprintf(w, "Hello World")
		})
		router.ListenAndServe()
	}()

	curl(t, "OPTIONS", "http://localhost:60310/admin/user/info", []byte("Forbidden\n"))
}

func TestServeFiles(t *testing.T) {
	go func() {
		router := middleware.New()
		router.Logger = logger
		router.Port = 60311
		defer router.Shutdown()
		router.STATIC(".", "/cdn")
		router.ListenAndServe()
	}()

	data, err := ioutil.ReadFile("LICENSE.md")

	if err != nil {
		t.Fatalf("cannot read LICENSE.md %s", err)
		return
	}

	curl(t, "GET", "http://localhost:60311/cdn/LICENSE.md", data)
}

func TestServeFilesFake(t *testing.T) {
	go func() {
		router := middleware.New()
		router.Logger = logger
		router.Port = 60335
		defer router.Shutdown()
		router.GET("/updates/appcast.xml", func(w http.ResponseWriter, r *http.Request) {
			fmt.Fprintf(w, "<xml></xml>")
		})
		router.ListenAndServe()
	}()

	curl(t, "GET", "http://localhost:60335/updates/appcast.xml", []byte("<xml></xml>"))
}

func TestServeFilesFakeScript(t *testing.T) {
	go func() {
		router := middleware.New()
		router.Logger = logger
		router.Port = 60336
		defer router.Shutdown()
		router.GET("/tag/js/gpt.js", func(w http.ResponseWriter, r *http.Request) {
			fmt.Fprintf(w, "(function(E){})")
		})
		router.ListenAndServe()
	}()

	curl(t, "GET", "http://localhost:60336/tag/js/gpt.js", []byte("(function(E){})"))
	curl(t, "GET", "http://localhost:60336/tag/js/foo.js", []byte("404 page not found\n"))
}

func TestTrailingSlash(t *testing.T) {
	go func() {
		router := middleware.New()
		router.Logger = logger
		router.Port = 60313
		defer router.Shutdown()
		router.GET("/hello/world/", func(w http.ResponseWriter, r *http.Request) {
			fmt.Fprintf(w, "Hello World")
		})
		router.ListenAndServe()
	}()

	curl(t, "GET", "http://localhost:60313/hello/world/", []byte("Hello World"))
}

func TestTrailingSlashDynamic(t *testing.T) {
	go func() {
		router := middleware.New()
		router.Logger = logger
		router.Port = 60316
		defer router.Shutdown()
		router.POST("/api/:id/store/", func(w http.ResponseWriter, r *http.Request) {
			fmt.Fprintf(w, "store")
		})
		router.ListenAndServe()
	}()

	curl(t, "POST", "http://localhost:60316/api/123/store/", []byte("store"))
}

func TestTrailingSlashDynamicMultiple(t *testing.T) {
	go func() {
		router := middleware.New()
		router.Logger = logger
		router.Port = 60324
		defer router.Shutdown()
		router.POST("/api/:id/store/", func(w http.ResponseWriter, r *http.Request) {
			fmt.Fprintf(w, "dynamic")
		})
		router.ListenAndServe()
	}()

	curl(t, "POST", "http://localhost:60324/api/123/////store/", []byte("dynamic"))
}

func TestMultipleRoutes(t *testing.T) {
	go func() {
		router := middleware.New()
		router.Logger = logger
		router.Port = 60315
		defer router.Shutdown()
		router.GET("/hello/world/", func(w http.ResponseWriter, r *http.Request) {
			fmt.Fprintf(w, "Hello World")
		})
		router.GET("/lorem/ipsum/", func(w http.ResponseWriter, r *http.Request) {
			fmt.Fprintf(w, "Lorem Ipsum")
		})
		router.ListenAndServe()
	}()

	curl(t, "GET", "http://localhost:60315/hello/world/", []byte("Hello World"))
	curl(t, "GET", "http://localhost:60315/lorem/ipsum/", []byte("Lorem Ipsum"))
}

func TestMultipleDynamic(t *testing.T) {
	go func() {
		router := middleware.New()
		router.Logger = logger
		router.Port = 60332
		defer router.Shutdown()
		router.GET("/hello/:first/:last/info", func(w http.ResponseWriter, r *http.Request) {
			fmt.Fprintf(
				w,
				"Hello %s %s",
				middleware.Param(r, "first"),
				middleware.Param(r, "last"),
			)
		})
		router.ListenAndServe()
	}()

	curl(t, "GET", "http://localhost:60332/hello/john/smith/info", []byte("Hello john smith"))
}
