package middleware_test

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"net/http"
	"testing"

	"github.com/cixtor/middleware"
)

func curl(method string, target string) ([]byte, error) {
	req, err := http.NewRequest(method, target, nil)

	if err != nil {
		return nil, err
	}

	res, err := http.DefaultClient.Do(req)

	if err != nil {
		return nil, err
	}

	defer res.Body.Close()

	data, err := ioutil.ReadAll(res.Body)

	if err != nil {
		return nil, err
	}

	return data, nil
}

func TestIndex(t *testing.T) {
	go func() {
		router := middleware.New()
		router.Port = "58302"
		router.GET("/foobar", func(w http.ResponseWriter, r *http.Request) {
			fmt.Fprintf(w, "Hello World")
		})
		router.ListenAndServe()
	}()

	data, err := curl("GET", "http://localhost:58302/foobar")

	if err != nil {
		t.Fatalf("curl %s", err)
		return
	}

	if !bytes.Equal(data, []byte("Hello World")) {
		t.Fatal("GET / request failure")
		return
	}
}

func TestPOST(t *testing.T) {
	go func() {
		router := middleware.New()
		router.Port = "58303"
		router.POST("/foobar", func(w http.ResponseWriter, r *http.Request) {
			fmt.Fprintf(w, "Hello World POST")
		})
		router.ListenAndServe()
	}()

	data, err := curl("POST", "http://localhost:58303/foobar")

	if err != nil {
		t.Fatalf("curl %s", err)
		return
	}

	if !bytes.Equal(data, []byte("Hello World POST")) {
		t.Fatal("POST /foobar request failure")
		return
	}
}
