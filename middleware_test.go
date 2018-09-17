package middleware_test

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"net/http"
	"testing"

	"github.com/cixtor/middleware"
)

func TestIndex(t *testing.T) {
	go func() {
		router := middleware.New()
		router.Port = "58302"
		router.GET("/", func(w http.ResponseWriter, r *http.Request) {
			fmt.Fprintf(w, "Hello World")
		})
		router.ListenAndServe()
	}()

	res, err := http.Get("http://localhost:58302/")

	if err != nil {
		t.Fatal(err)
		return
	}

	defer res.Body.Close()

	data, err := ioutil.ReadAll(res.Body)

	if err != nil {
		t.Fatal(err)
		return
	}

	if !bytes.Equal(data, []byte("Hello World")) {
		t.Fatal("response for index page was incorrect")
		return
	}
}
