package middleware_test

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"net/http"
	"testing"

	"github.com/cixtor/middleware"
)

func curl(t *testing.T, method string, target string, expected []byte) {
	req, err := http.NewRequest(method, target, nil)

	if err != nil {
		t.Fatalf("http.NewRequest %s", err)
		return
	}

	res, err := http.DefaultClient.Do(req)

	if err != nil {
		t.Fatalf("http.DefaultClient %s", err)
		return
	}

	defer res.Body.Close()

	data, err := ioutil.ReadAll(res.Body)

	if err != nil {
		t.Fatalf("ioutil.ReadAll %s", err)
		return
	}

	if !bytes.Equal(data, expected) {
		t.Fatalf("%s %s\nexpected: %q\nreceived: %q", method, target, expected, data)
		return
	}
}

func TestIndex(t *testing.T) {
	go func() {
		router := middleware.New()
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
		router.Port = "58304"
		defer router.Shutdown()
		router.GET("/", func(w http.ResponseWriter, r *http.Request) {
			fmt.Fprintf(w, "Hello World GET")
		})
		router.ListenAndServe()
	}()

	curl(t, "GET", "http://localhost:58304/notfound", []byte("404 page not found\n"))
}

func TestDirectoryListing(t *testing.T) {
	go func() {
		router := middleware.New()
		router.Port = "58305"
		defer router.Shutdown()
		router.STATIC(".", "/assets")
		router.ListenAndServe()
	}()

	curl(t, "GET", "http://localhost:58305/assets/images/", []byte("Forbidden\n"))
}
