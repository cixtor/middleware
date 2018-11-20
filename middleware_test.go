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
		router.Port = "58302"
		defer router.Shutdown()
		router.GET("/foobar", func(w http.ResponseWriter, r *http.Request) {
			fmt.Fprintf(w, "Hello World")
		})
		router.ListenAndServe()
	}()

	curl(t, "GET", "http://localhost:58302/foobar", []byte("Hello World"))
}

func TestPOST(t *testing.T) {
	go func() {
		router := middleware.New()
		router.Logger = logger
		router.Port = "58303"
		defer router.Shutdown()
		router.POST("/foobar", func(w http.ResponseWriter, r *http.Request) {
			fmt.Fprintf(w, "Hello World POST")
		})
		router.ListenAndServe()
	}()

	curl(t, "POST", "http://localhost:58303/foobar", []byte("Hello World POST"))
}

func TestNotFound(t *testing.T) {
	go func() {
		router := middleware.New()
		router.Logger = logger
		router.Port = "58304"
		defer router.Shutdown()
		router.GET("/", func(w http.ResponseWriter, r *http.Request) {
			fmt.Fprintf(w, "Hello World GET")
		})
		router.ListenAndServe()
	}()

	curl(t, "GET", "http://localhost:58304/notfound", []byte("404 page not found\n"))
}

func TestNotFoundSimilar(t *testing.T) {
	go func() {
		router := middleware.New()
		router.Logger = logger
		router.Port = "58314"
		defer router.Shutdown()
		router.GET("/lorem/ipsum/dolor", func(w http.ResponseWriter, r *http.Request) {
			fmt.Fprintf(w, "Hello World GET")
		})
		router.ListenAndServe()
	}()

	curl(t, "GET", "http://localhost:58314/dolor/ipsum/lorem", []byte("404 page not found\n"))
}

func TestDirectoryListing(t *testing.T) {
	go func() {
		router := middleware.New()
		router.Logger = logger
		router.Port = "58305"
		defer router.Shutdown()
		router.STATIC(".", "/assets")
		router.ListenAndServe()
	}()

	curl(t, "GET", "http://localhost:58305/assets", []byte("Forbidden\n"))
	curl(t, "GET", "http://localhost:58305/assets/", []byte("Forbidden\n"))
	curl(t, "GET", "http://localhost:58305/assets/.git", []byte("Forbidden\n"))
	curl(t, "GET", "http://localhost:58305/assets/.git/", []byte("Forbidden\n"))
	curl(t, "GET", "http://localhost:58305/assets/.git/objects", []byte("Forbidden\n"))
	curl(t, "GET", "http://localhost:58305/assets/.git/objects/", []byte("Forbidden\n"))
}

func TestSingleParam(t *testing.T) {
	go func() {
		router := middleware.New()
		router.Logger = logger
		router.Port = "58306"
		defer router.Shutdown()
		router.PUT("/hello/:name", func(w http.ResponseWriter, r *http.Request) {
			fmt.Fprintf(w, "Hello %s", middleware.Param(r, "name"))
		})
		router.ListenAndServe()
	}()

	curl(t, "PUT", "http://localhost:58306/hello/john", []byte("Hello john"))
}

func TestMultiParam(t *testing.T) {
	go func() {
		router := middleware.New()
		router.Logger = logger
		router.Port = "58307"
		defer router.Shutdown()
		router.PATCH("/:group/:section", func(w http.ResponseWriter, r *http.Request) {
			group := middleware.Param(r, "group")
			section := middleware.Param(r, "section")
			fmt.Fprintf(w, "Page /%s/%s", group, section)
		})
		router.ListenAndServe()
	}()

	curl(t, "PATCH", "http://localhost:58307/account/info", []byte("Page /account/info"))
}

func TestMultiParamPrefix(t *testing.T) {
	go func() {
		router := middleware.New()
		router.Logger = logger
		router.Port = "58308"
		defer router.Shutdown()
		router.DELETE("/foo/:group/:section", func(w http.ResponseWriter, r *http.Request) {
			group := middleware.Param(r, "group")
			section := middleware.Param(r, "section")
			fmt.Fprintf(w, "Page /foo/%s/%s", group, section)
		})
		router.ListenAndServe()
	}()

	curl(t, "DELETE", "http://localhost:58308/foo/account/info", []byte("Page /foo/account/info"))
}

func TestComplexParam(t *testing.T) {
	go func() {
		router := middleware.New()
		router.Logger = logger
		router.Port = "58312"
		defer router.Shutdown()
		router.PUT("/account/:name/info", func(w http.ResponseWriter, r *http.Request) {
			fmt.Fprintf(w, "Hello %s", middleware.Param(r, "name"))
		})
		router.ListenAndServe()
	}()

	curl(t, "PUT", "http://localhost:58312/account/john/info", []byte("Hello john"))
}

func TestAllowAccess(t *testing.T) {
	go func() {
		router := middleware.New()
		router.Logger = logger
		router.Port = "58309"
		defer router.Shutdown()
		router.AllowAccessExcept([]string{"[::1]"})
		router.OPTIONS("/admin/user/info", func(w http.ResponseWriter, r *http.Request) {
			fmt.Fprintf(w, "Hello World")
		})
		router.ListenAndServe()
	}()

	curl(t, "OPTIONS", "http://localhost:58309/admin/user/info", []byte("Forbidden\n"))
}

func TestDenyAccess(t *testing.T) {
	go func() {
		router := middleware.New()
		router.Logger = logger
		router.Port = "58310"
		defer router.Shutdown()
		router.DenyAccessExcept([]string{"82.82.82.82"})
		router.OPTIONS("/admin/user/info", func(w http.ResponseWriter, r *http.Request) {
			fmt.Fprintf(w, "Hello World")
		})
		router.ListenAndServe()
	}()

	curl(t, "OPTIONS", "http://localhost:58310/admin/user/info", []byte("Forbidden\n"))
}

func TestServeFiles(t *testing.T) {
	go func() {
		router := middleware.New()
		router.Logger = logger
		router.Port = "58311"
		defer router.Shutdown()
		router.STATIC(".", "/cdn")
		router.ListenAndServe()
	}()

	data, err := ioutil.ReadFile("LICENSE.md")

	if err != nil {
		t.Fatalf("cannot read LICENSE.md %s", err)
		return
	}

	curl(t, "GET", "http://localhost:58311/cdn/LICENSE.md", data)
}

func TestTrailingSlash(t *testing.T) {
	go func() {
		router := middleware.New()
		router.Logger = logger
		router.Port = "58313"
		defer router.Shutdown()
		router.GET("/hello/world/", func(w http.ResponseWriter, r *http.Request) {
			fmt.Fprintf(w, "Hello World")
		})
		router.ListenAndServe()
	}()

	curl(t, "GET", "http://localhost:58313/hello/world/", []byte("Hello World"))
}

func TestMultipleRoutes(t *testing.T) {
	go func() {
		router := middleware.New()
		router.Logger = logger
		router.Port = "58315"
		defer router.Shutdown()
		router.GET("/hello/world/", func(w http.ResponseWriter, r *http.Request) {
			fmt.Fprintf(w, "Hello World")
		})
		router.GET("/lorem/ipsum/", func(w http.ResponseWriter, r *http.Request) {
			fmt.Fprintf(w, "Lorem Ipsum")
		})
		router.ListenAndServe()
	}()

	curl(t, "GET", "http://localhost:58315/hello/world/", []byte("Hello World"))
	curl(t, "GET", "http://localhost:58315/lorem/ipsum/", []byte("Lorem Ipsum"))
}
