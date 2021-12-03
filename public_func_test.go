package middleware_test

import (
	"bytes"
	"errors"
	"io/ioutil"
	"net"
	"net/http"
	"net/url"
	"os"
	"syscall"
	"testing"
	"time"

	"github.com/cixtor/middleware"
)

// newTestServer returns a server and an ephemeral port to listen.
func newTestServer(t *testing.T) (*middleware.Middleware, net.Addr) {
	srv := middleware.New()

	addr, err := srv.FreePort()

	if err != nil {
		t.Fatal("shu", err)
	}

	return srv, addr
}

func curl(t *testing.T, method string, host string, addr net.Addr, endpoint string, expected []byte) {
	target := "http://" + addr.String() + endpoint
	req, err := http.NewRequest(method, target, nil)

	if err != nil {
		t.Fatalf("http.NewRequest %s", err)
		return
	}

	req.Host = host

	// It looks like the TCP resolver that Middleware leverages to obtain a
	// random free port delays the execution of the server listener long enough
	// for this function to randomly fail 36.95% of the time when the tests run
	// all at the same time.
	//
	// I managed to fix this by delaying the execution of the HTTP request a
	// couple of milliseconds. A rudimentary benchmark with 10 iterations as a
	// warmup returns the following results:
	//
	//   > Benchmark 1: go test
	//   >   Time (mean ± σ):     949.3 ms ±  19.5 ms    [User: 722.2 ms, System: 467.7 ms]
	//   >   Range (min … max):   904.9 ms … 968.8 ms    10 runs
	//
	// TODO: find a way to execute all the tests without the 2ms cURL delay.
	time.Sleep(time.Millisecond * 2)

	res, err := http.DefaultClient.Do(req)

	if err != nil {
		t.Fatalf("http.DefaultClient %s", err)
		return
	}

	defer res.Body.Close()

	out, err := ioutil.ReadAll(res.Body)

	if err != nil {
		t.Fatalf("ioutil.ReadAll %s", err)
		return
	}

	if !bytes.Equal(out, expected) {
		t.Fatalf("%s %s\nexpected: %q\nreceived: %q", method, target, expected, out)
		return
	}
}

func shouldNotCurl(t *testing.T, method string, host string, addr net.Addr, endpoint string) {
	target := "http://" + addr.String() + endpoint
	req, err := http.NewRequest(method, target, nil)

	if err != nil {
		t.Fatalf("http.NewRequest %s", err)
		return
	}

	req.Host = host

	if _, err := http.DefaultClient.Do(req); errors.Is(err, syscall.ECONNREFUSED) {
		// Detect "connection refused" error and return as a successful call.
		return
	}

	t.Fatalf("%s %s should have failed", method, target)
}

func TestIndex(t *testing.T) {
	srv, addr := newTestServer(t)
	srv.DiscardLogs()
	defer srv.Shutdown()
	srv.GET("/foobar", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("Hello World"))
	})
	go srv.ListenAndServe(addr.String())

	curl(t, "GET", "localhost", addr, "/foobar", []byte("Hello World"))
}

func TestUse(t *testing.T) {
	srv, addr := newTestServer(t)

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

	srv.DiscardLogs()
	srv.Use(lorem)
	srv.Use(ipsum)
	srv.Use(dolor)
	defer srv.Shutdown()
	srv.GET("/foobar", func(w http.ResponseWriter, r *http.Request) {
		a := w.Header().Get("lorem")
		b := w.Header().Get("ipsum")
		c := w.Header().Get("dolor")
		w.Write([]byte(a + ":" + b + ":" + c))
	})
	go srv.ListenAndServe(addr.String())

	curl(t, "GET", "localhost", addr, "/foobar", []byte("lorem:ipsum:dolor"))
}

func TestUse2(t *testing.T) {
	srv, addr := newTestServer(t)

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

	srv.DiscardLogs()
	defer srv.Shutdown()
	srv.GET("/foobar", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("<4:foobar>"))
	})
	srv.Use(lorem)
	srv.Use(ipsum)
	srv.Use(dolor)
	go srv.ListenAndServe(addr.String())

	curl(t, "GET", "localhost", addr, "/foobar", []byte("<1:lorem><2:ipsum><3:dolor><4:foobar>"))
}

func TestPOST(t *testing.T) {
	srv, addr := newTestServer(t)
	srv.DiscardLogs()
	defer srv.Shutdown()
	srv.POST("/foobar", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("Hello World POST"))
	})
	go srv.ListenAndServe(addr.String())

	curl(t, "POST", "localhost", addr, "/foobar", []byte("Hello World POST"))
}

func TestNotFound(t *testing.T) {
	srv, addr := newTestServer(t)
	srv.DiscardLogs()
	defer srv.Shutdown()
	srv.GET("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("Hello World GET"))
	})
	go srv.ListenAndServe(addr.String())

	curl(t, "GET", "localhost", addr, "/notfound", []byte("404 page not found\n"))
}

func TestNotFoundSimilar(t *testing.T) {
	srv, addr := newTestServer(t)
	srv.DiscardLogs()
	defer srv.Shutdown()
	srv.GET("/lorem/ipsum/dolor", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("Hello World GET"))
	})
	go srv.ListenAndServe(addr.String())

	curl(t, "GET", "localhost", addr, "/lorem/ipsum/dolores", []byte("404 page not found\n"))
}

func TestNotFoundInvalid(t *testing.T) {
	srv, addr := newTestServer(t)
	srv.DiscardLogs()
	srv.NotFound = nil
	defer srv.Shutdown()
	srv.GET("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("Hello World GET"))
	})
	go srv.ListenAndServe(addr.String())

	curl(t, "GET", "localhost", addr, "/test", []byte("404 page not found\n"))
}

func TestNotFoundCustom(t *testing.T) {
	srv, addr := newTestServer(t)
	srv.DiscardLogs()
	srv.NotFound = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("404 page does not exist"))
	})
	defer srv.Shutdown()
	srv.GET("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("Hello World GET"))
	})
	go srv.ListenAndServe(addr.String())

	curl(t, "GET", "localhost", addr, "/test", []byte("404 page does not exist"))
}

func TestNotFoundCustomWithInterceptor(t *testing.T) {
	srv, addr := newTestServer(t)
	srv.DiscardLogs()
	srv.NotFound = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("404 missing page\n"))
	})
	srv.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/test" {
				w.Write([]byte("hello interceptor\n"))
				// return /* do not return */
			}
			next.ServeHTTP(w, r)
		})
	})
	defer srv.Shutdown()
	srv.GET("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("Hello World GET"))
	})
	go srv.ListenAndServe(addr.String())

	curl(t, "GET", "localhost", addr, "/test", []byte("hello interceptor\n404 missing page\n"))
}

func TestDirectoryListing(t *testing.T) {
	srv, addr := newTestServer(t)
	srv.DiscardLogs()
	defer srv.Shutdown()
	srv.STATIC(".", "/assets")
	go srv.ListenAndServe(addr.String())

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
			curl(t, "GET", "localhost", addr, input[1], []byte(input[2]))
		})
	}
}

func TestSingleParam(t *testing.T) {
	srv, addr := newTestServer(t)
	srv.DiscardLogs()
	defer srv.Shutdown()
	srv.PUT("/hello/:name", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(middleware.Param(r, "name")))
	})
	go srv.ListenAndServe(addr.String())

	curl(t, "PUT", "localhost", addr, "/hello/john", []byte("john"))
}

func TestMultiParam(t *testing.T) {
	srv, addr := newTestServer(t)
	srv.DiscardLogs()
	defer srv.Shutdown()
	srv.PATCH("/:group/:section", func(w http.ResponseWriter, r *http.Request) {
		group := middleware.Param(r, "group")
		section := middleware.Param(r, "section")
		w.Write([]byte("page /" + group + "/" + section))
	})
	go srv.ListenAndServe(addr.String())

	curl(t, "PATCH", "localhost", addr, "/account/info", []byte("page /account/info"))
}

func TestMultiParamPrefix(t *testing.T) {
	srv, addr := newTestServer(t)
	srv.DiscardLogs()
	defer srv.Shutdown()
	srv.DELETE("/foo/:group/:section", func(w http.ResponseWriter, r *http.Request) {
		group := middleware.Param(r, "group")
		section := middleware.Param(r, "section")
		w.Write([]byte("page /foo/" + group + "/" + section))
	})
	go srv.ListenAndServe(addr.String())

	curl(t, "DELETE", "localhost", addr, "/foo/account/info", []byte("page /foo/account/info"))
}

func TestComplexParam(t *testing.T) {
	srv, addr := newTestServer(t)
	srv.DiscardLogs()
	defer srv.Shutdown()
	srv.PUT("/account/:name/info", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(middleware.Param(r, "name")))
	})
	go srv.ListenAndServe(addr.String())

	curl(t, "PUT", "localhost", addr, "/account/alice/info", []byte("alice"))
}

func TestServeFiles(t *testing.T) {
	srv, addr := newTestServer(t)
	srv.DiscardLogs()
	defer srv.Shutdown()
	srv.STATIC(".", "/cdn")
	go srv.ListenAndServe(addr.String())

	data, err := ioutil.ReadFile("LICENSE.md")

	if err != nil {
		t.Fatalf("cannot read LICENSE.md %s", err)
		return
	}

	curl(t, "GET", "localhost", addr, "/cdn/LICENSE.md", data)
}

func TestServeFilesFake(t *testing.T) {
	srv, addr := newTestServer(t)
	srv.DiscardLogs()
	defer srv.Shutdown()
	srv.GET("/updates/appcast.xml", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("<xml></xml>"))
	})
	go srv.ListenAndServe(addr.String())

	curl(t, "GET", "localhost", addr, "/updates/appcast.xml", []byte("<xml></xml>"))
}

func TestServeFilesFakeScript(t *testing.T) {
	srv, addr := newTestServer(t)
	srv.DiscardLogs()
	defer srv.Shutdown()
	srv.GET("/tag/js/gpt.js", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("(function(E){})"))
	})
	go srv.ListenAndServe(addr.String())

	curl(t, "GET", "localhost", addr, "/tag/js/gpt.js", []byte("(function(E){})"))
	curl(t, "GET", "localhost", addr, "/tag/js/foo.js", []byte("404 page not found\n"))
}

func TestRouteWithExtraSlash(t *testing.T) {
	srv, addr := newTestServer(t)
	srv.DiscardLogs()
	defer srv.Shutdown()
	srv.GET("/hello/world", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("hello"))
	})
	go srv.ListenAndServe(addr.String())

	curl(t, "GET", "localhost", addr, "/hello///////world", []byte("hello"))
}

func TestRouteWithExtraSlash2(t *testing.T) {
	srv, addr := newTestServer(t)
	srv.DiscardLogs()
	defer srv.Shutdown()
	srv.GET("/hello/world", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("hello"))
	})
	go srv.ListenAndServe(addr.String())

	curl(t, "GET", "localhost", addr, "///////hello/world", []byte("hello"))
}

func TestTrailingSlash(t *testing.T) {
	srv, addr := newTestServer(t)
	srv.DiscardLogs()
	defer srv.Shutdown()
	srv.GET("/hello/world/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("Hello World"))
	})
	go srv.ListenAndServe(addr.String())

	curl(t, "GET", "localhost", addr, "/hello/world/", []byte("Hello World"))
}

func TestTrailingSlashDynamic(t *testing.T) {
	srv, addr := newTestServer(t)
	srv.DiscardLogs()
	defer srv.Shutdown()
	srv.POST("/api/:id/store/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("store"))
	})
	go srv.ListenAndServe(addr.String())

	curl(t, "POST", "localhost", addr, "/api/123/store/", []byte("store"))
}

func TestTrailingSlashDynamicMultiple(t *testing.T) {
	srv, addr := newTestServer(t)
	srv.DiscardLogs()
	defer srv.Shutdown()
	srv.POST("/api/:id/store/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("dynamic"))
	})
	go srv.ListenAndServe(addr.String())

	curl(t, "POST", "localhost", addr, "/api/123/////store/", []byte("dynamic"))
}

func TestMultipleRoutes(t *testing.T) {
	srv, addr := newTestServer(t)
	srv.DiscardLogs()
	defer srv.Shutdown()
	srv.GET("/hello/world/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("Hello World"))
	})
	srv.GET("/lorem/ipsum/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("Lorem Ipsum"))
	})
	go srv.ListenAndServe(addr.String())

	curl(t, "GET", "localhost", addr, "/hello/world/", []byte("Hello World"))
	curl(t, "GET", "localhost", addr, "/lorem/ipsum/", []byte("Lorem Ipsum"))
}

func TestRouteWithAsterisk(t *testing.T) {
	srv, addr := newTestServer(t)
	srv.DiscardLogs()
	defer srv.Shutdown()
	srv.GET("/home/users/*/ignored/sections", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("robot"))
	})
	go srv.ListenAndServe(addr.String())

	curl(t, "GET", "localhost", addr, "/home/users/a/b/root", []byte("robot"))
}

func TestMultipleDynamic(t *testing.T) {
	srv, addr := newTestServer(t)
	srv.DiscardLogs()
	defer srv.Shutdown()
	srv.GET("/hello/:first/:last/info", func(w http.ResponseWriter, r *http.Request) {
		first := middleware.Param(r, "first")
		last := middleware.Param(r, "last")
		w.Write([]byte("Hello " + first + " " + last))
	})
	go srv.ListenAndServe(addr.String())

	curl(t, "GET", "localhost", addr, "/hello/john/smith/info", []byte("Hello john smith"))
}

func TestMultipleHosts(t *testing.T) {
	srv, addr := newTestServer(t)
	srv.DiscardLogs()
	defer srv.Shutdown()
	srv.Host("foo.test").GET("/hello/:name", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("@foo.test:" + middleware.Param(r, "name")))
	})
	srv.Host("bar.test").GET("/hello/:name", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("@bar.test:" + middleware.Param(r, "name")))
	})
	go srv.ListenAndServe(addr.String())

	curl(t, "GET", "foo.test", addr, "/hello/john", []byte("@foo.test:john"))
	curl(t, "GET", "bar.test", addr, "/hello/alice", []byte("@bar.test:alice"))
}

func TestDefaultHost(t *testing.T) {
	srv, addr := newTestServer(t)
	srv.DiscardLogs()
	defer srv.Shutdown()
	srv.GET("/hello/:name", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("Hello " + middleware.Param(r, "name")))
	})
	srv.Host("foo.test").GET("/world/:name", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("World " + middleware.Param(r, "name")))
	})
	go srv.ListenAndServe(addr.String())

	curl(t, "GET", "localhost", addr, "/hello/john", []byte("Hello john"))
	curl(t, "GET", "foo.test", addr, "/world/earth", []byte("World earth"))
	curl(t, "GET", "bar.test", addr, "/anything", []byte("404 page not found\n"))
}

func TestMethodHandle(t *testing.T) {
	srv, addr := newTestServer(t)
	srv.DiscardLogs()
	defer srv.Shutdown()
	srv.Handle("HELLOWORLD", "/foobar", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("Hello World"))
	})
	go srv.ListenAndServe(addr.String())

	curl(t, "HELLOWORLD", "localhost", addr, "/foobar", []byte("Hello World"))
}

func TestMethodCONNECT(t *testing.T) {
	srv, addr := newTestServer(t)
	srv.DiscardLogs()
	defer srv.Shutdown()
	srv.CONNECT("/foobar", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("Hello World"))
	})
	go srv.ListenAndServe(addr.String())

	curl(t, "CONNECT", "localhost", addr, "/foobar", []byte("Hello World"))
}

func TestMethodTRACE(t *testing.T) {
	srv, addr := newTestServer(t)
	srv.DiscardLogs()
	defer srv.Shutdown()
	srv.TRACE("/foobar", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("Hello World"))
	})
	go srv.ListenAndServe(addr.String())

	curl(t, "TRACE", "localhost", addr, "/foobar", []byte("Hello World"))
}

func TestMethodCOPY(t *testing.T) {
	srv, addr := newTestServer(t)
	srv.DiscardLogs()
	defer srv.Shutdown()
	srv.COPY("/foobar", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("Hello World"))
	})
	go srv.ListenAndServe(addr.String())

	curl(t, "COPY", "localhost", addr, "/foobar", []byte("Hello World"))
}

func TestMethodLOCK(t *testing.T) {
	srv, addr := newTestServer(t)
	srv.DiscardLogs()
	defer srv.Shutdown()
	srv.LOCK("/foobar", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("Hello World"))
	})
	go srv.ListenAndServe(addr.String())

	curl(t, "LOCK", "localhost", addr, "/foobar", []byte("Hello World"))
}

func TestMethodMKCOL(t *testing.T) {
	srv, addr := newTestServer(t)
	srv.DiscardLogs()
	defer srv.Shutdown()
	srv.MKCOL("/foobar", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("Hello World"))
	})
	go srv.ListenAndServe(addr.String())

	curl(t, "MKCOL", "localhost", addr, "/foobar", []byte("Hello World"))
}

func TestMethodMOVE(t *testing.T) {
	srv, addr := newTestServer(t)
	srv.DiscardLogs()
	defer srv.Shutdown()
	srv.MOVE("/foobar", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("Hello World"))
	})
	go srv.ListenAndServe(addr.String())

	curl(t, "MOVE", "localhost", addr, "/foobar", []byte("Hello World"))
}

func TestMethodPROPFIND(t *testing.T) {
	srv, addr := newTestServer(t)
	srv.DiscardLogs()
	defer srv.Shutdown()
	srv.PROPFIND("/foobar", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("Hello World"))
	})
	go srv.ListenAndServe(addr.String())

	curl(t, "PROPFIND", "localhost", addr, "/foobar", []byte("Hello World"))
}

func TestMethodPROPPATCH(t *testing.T) {
	srv, addr := newTestServer(t)
	srv.DiscardLogs()
	defer srv.Shutdown()
	srv.PROPPATCH("/foobar", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("Hello World"))
	})
	go srv.ListenAndServe(addr.String())

	curl(t, "PROPPATCH", "localhost", addr, "/foobar", []byte("Hello World"))
}

func TestMethodUNLOCK(t *testing.T) {
	srv, addr := newTestServer(t)
	srv.DiscardLogs()
	defer srv.Shutdown()
	srv.UNLOCK("/foobar", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("Hello World"))
	})
	go srv.ListenAndServe(addr.String())

	curl(t, "UNLOCK", "localhost", addr, "/foobar", []byte("Hello World"))
}

func TestEndpointOrder(t *testing.T) {
	srv, addr := newTestServer(t)
	srv.DiscardLogs()
	defer srv.Shutdown()
	srv.GET("/*", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("endpoint #7"))
	})
	srv.GET("/help/:group/:question/*", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("endpoint #6"))
	})
	srv.GET("/usr/local/:group/:package/*", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("endpoint #5"))
	})
	srv.GET("/user/:userid", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("endpoint #4"))
	})
	srv.GET("/auth/:sso", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("endpoint #3"))
	})
	srv.GET("/blog/:name/slug", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("endpoint #2"))
	})
	srv.GET("/help/:page/:group/comments", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("endpoint #1"))
	})
	go srv.ListenAndServe(addr.String())

	inputs := [][]string{
		{"test_0", "/help/viva/family/comments", "endpoint #1"},
		{"test_1", "/blog/hello-world/slug", "endpoint #2"},
		{"test_2", "/auth/openid", "endpoint #3"},
		{"test_3", "/user/12345", "endpoint #4"},
		{"test_4", "/usr/local/etc/openssl/cert.pem", "endpoint #5"},
		{"test_5", "/help/viva/family/foo/bar", "endpoint #6"},
		{"test_6", "/help/viva/family/foobar", "endpoint #6"},
		{"test_7", "/x/help/viva/family", "endpoint #7"},
		{"test_8", "/y/any/thing", "endpoint #7"},
		{"test_9", "/z/hello/world/how/are/you", "endpoint #7"},
	}

	for _, input := range inputs {
		t.Run(input[0], func(t *testing.T) {
			curl(t, "GET", "localhost", addr, ""+input[1], []byte(input[2]))
		})
	}
}

func TestAmbiguousPath(t *testing.T) {
	srv, addr := newTestServer(t)
	srv.DiscardLogs()
	defer srv.Shutdown()
	srv.GET("/:package", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("package"))
	})
	srv.GET("/:package/-/:archive", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("package/archive"))
	})
	go srv.ListenAndServe(addr.String())

	curl(t, "GET", "localhost", addr, "/foobar", []byte("package"))
	curl(t, "GET", "localhost", addr, "/foobar/-/foobar.tgz", []byte("package/archive"))
}

func TestAmbiguousPath2(t *testing.T) {
	srv, addr := newTestServer(t)
	srv.DiscardLogs()
	defer srv.Shutdown()
	srv.GET("/:package", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("package"))
	})
	srv.GET("/:module/:package", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("module/package"))
	})
	srv.GET("/:package/-/:archive", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("package/archive"))
	})
	go srv.ListenAndServe(addr.String())

	curl(t, "GET", "localhost", addr, "/foobar", []byte("package"))
	curl(t, "GET", "localhost", addr, "/@babel/core", []byte("module/package"))
	curl(t, "GET", "localhost", addr, "/foobar/-/foobar.tgz", []byte("package/archive"))
}

func TestAmbiguousPath3(t *testing.T) {
	srv, addr := newTestServer(t)
	srv.DiscardLogs()
	defer srv.Shutdown()
	srv.GET("/usr/local/:group/:user/*", func(w http.ResponseWriter, r *http.Request) {
		group := middleware.Param(r, "group")
		user := middleware.Param(r, "user")
		w.Write([]byte(group + ":" + user))
	})
	go srv.ListenAndServe(addr.String())

	inputs := [][]string{
		{"should not exist 1", "/usr/local", "404 page not found\n"},
		{"should not exist 2", "/usr/local/etc", "404 page not found\n"},
		{"should not exist 3", "/usr/local/etc/openssl", "404 page not found\n"},
		{"should exist", "/usr/local/etc/openssl/cert.pem", "etc:openssl"},
	}

	for _, input := range inputs {
		t.Run(input[0], func(t *testing.T) {
			curl(t, "GET", "localhost", addr, ""+input[1], []byte(input[2]))
		})
	}
}

type telemetry struct {
	called bool
	latest middleware.AccessLog
}

func (t telemetry) ListeningOn(addr net.Addr) {}

func (t telemetry) Shutdown(err error) {}

func (t *telemetry) Log(data middleware.AccessLog) {
	t.called = true
	t.latest = data
}

func TestResponseCallback(t *testing.T) {
	srv, addr := newTestServer(t)
	tracer := &telemetry{}
	srv.Logger = tracer
	defer srv.Shutdown()
	srv.GET("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("Hello World"))
	})
	go srv.ListenAndServe(addr.String())

	curl(t, "GET", "localhost", addr, "/?hello=world&foo=bar", []byte("Hello World"))

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
	srv, addr := newTestServer(t)
	srv.DiscardLogs()
	srv.GET("/s", func(w http.ResponseWriter, r *http.Request) { w.Write([]byte("xyz")) })

	go srv.ListenAndServe(addr.String())

	curl(t, "GET", "localhost", addr, "/s", []byte("xyz"))

	srv.Shutdown()

	shouldNotCurl(t, "GET", "localhost", addr, "/s")
}

type CustomSignal int

func (CustomSignal) Signal() {}

func (CustomSignal) String() string { return "custom signal" }

func TestShutdownWithChannel(t *testing.T) {
	srv, addr := newTestServer(t)

	done := false
	quit := make(chan os.Signal, 1)
	next := make(chan bool, 1)

	srv.DiscardLogs()
	srv.GET("/swc", func(w http.ResponseWriter, r *http.Request) { w.Write([]byte("abc")) })

	go func() {
		<-quit
		srv.Shutdown()
		next <- true
		done = true
	}()

	go srv.ListenAndServe(addr.String())

	curl(t, "GET", "localhost", addr, "/swc", []byte("abc"))
	quit <- CustomSignal(60310) // Call middleware.Shutdown to stop the server.

	<-next // Wait for middleware.Shutdown to finish.
	shouldNotCurl(t, "GET", "localhost", addr, "/swc")

	if !done {
		t.Fatal("goroutine with middleware.Shutdown did not run correctly")
	}
}

func TestShutdownAddon(t *testing.T) {
	srv, addr := newTestServer(t)
	done := false
	srv.DiscardLogs()
	srv.OnShutdown = func() { done = true }
	srv.GET("/s", func(w http.ResponseWriter, r *http.Request) { w.Write([]byte("XD")) })

	go srv.ListenAndServe(addr.String())

	curl(t, "GET", "localhost", addr, "/s", []byte("XD"))
	srv.Shutdown()
	shouldNotCurl(t, "GET", "localhost", addr, "/s")

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

func (l LoggerAndNewLines) ListeningOn(addr net.Addr) {}

func (l LoggerAndNewLines) Shutdown(err error) {}

func (l *LoggerAndNewLines) Log(data middleware.AccessLog) {
	l.metadata = data
}

func TestLoggerAndNewLines(t *testing.T) {
	srv, addr := newTestServer(t)
	logger := &LoggerAndNewLines{}
	srv.Logger = logger
	defer srv.Shutdown()
	go srv.ListenAndServe(addr.String())

	curl(t, "GET", "localhost", addr, "/foo%0abar", []byte("Method Not Allowed\n"))

	expected := `"GET /foo\nbar HTTP/1.1"`

	if str := logger.metadata.Request(); str != expected {
		t.Fatalf("incorrect request section in access log:\n- %s\n+ %s", expected, str)
	}
}
