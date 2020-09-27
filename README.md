# Go HTTP Middleware [![GoReport](https://goreportcard.com/badge/github.com/cixtor/middleware)](https://goreportcard.com/report/github.com/cixtor/middleware) [![GoDoc](https://godoc.org/github.com/cixtor/middleware?status.svg)](https://godoc.org/github.com/cixtor/middleware)

HTTP middleware for web services [written in Go](https://golang.org/) _(aka. Golang)_.

* Follows Go standards to satisfy the [http.HandlerFunc](https://golang.org/pkg/net/http/#HandlerFunc) interface
* Supports different HTTP methods _(GET, POST, OPTIONS, etc.)_
* Provides a static file server _(CSS, JavaScript, Images, etc.)_
* Offers options to configure timeouts _(idle, read, write, shutdown, etc.)_
* Prevents directory listing attacks on endpoints with no index file
* Supports multiple hostnames in the same web server
* Supports dynamic named parameters in the URL
* Supports custom "404 Not Found" pages

## Installation

```sh
go get -u github.com/cixtor/middleware
```

## Usage

Below is a basic example:

```golang
package main

import (
    "log"
    "github.com/cixtor/middleware"
)

func main() {
    srv := middleware.New()
    srv.GET("/", index)
    log.Fatal(srv.ListenAndServe(":3000"))
}

func index(w http.ResponseWriter, r *http.Request) {
    _, _ = w.Write([]byte("Hello World!\n"))
}
```

## Sane Timeouts

By default, all the timeouts are configured to five seconds, you can change them like this:

```golang
srv.IdleTimeout       = time.Second * 10
srv.ReadTimeout       = time.Second * 10
srv.WriteTimeout      = time.Second * 10
srv.ShutdownTimeout   = time.Second * 10
srv.ReadHeaderTimeout = time.Second * 10
```

## Serving Static Files

```golang
srv.STATIC("/var/www/public_html", "/assets")
```

In the example above, we are assumming that a directory located at `/var/www/public_html/` exists. With that in mind, every request to an URL with the `/assets/` prefix will be handled by the `http.ServeFiles()` method as long as the requested resource is pointing to an existing file.

A request to a directory returns "403 Forbidden" to prevent directory listing attacks.

A request to a nonexistent file returns "404 Not Found".

## Graceful Shutdown

A graceful shutdown can be added with the following code:

```golang
import (
    "fmt"
    "os"
    "os/signal"
    "syscall"
}

func main() {
    shutdown := make(chan os.Signal, 1)
    signal.Notify(shutdown, syscall.SIGINT, syscall.SIGTERM)

    go func() {
        <-shutdown
        srv.Shutdown()
        […] // close resources.
        fmt.Println("finished")
    }()

    log.Fatal(srv.ListenAndServe(":3000"))
}
```

## TLS Support

Generate the SSL certificates:

```sh
openssl genrsa -out server.key 2048

openssl ecparam -genkey -name secp384r1 -out server.key
# Country Name (2 letter code) []:CA
# State or Province Name (full name) []:British Columbia
# Locality Name (eg, city) []:Vancouver
# Organization Name (eg, company) []:Foobar Inc.
# Organizational Unit Name (eg, section) []:
# Common Name (eg, fully qualified host name) []:middleware.test
# Email Address []:webmaster@middleware.test

echo -e "127.0.0.1 middleware.test" | sudo tee -a /etc/hosts
```

Use `srv.ListenAndServeTLS(":8080", "server.crt", "server.key", nil)` to start the web server.

Test the connection using cURL `curl --cacert server.crt "https://middleware.test:8080"`

Add a custom TLS configuration by passing a `&tls.Config{}` as the last parameter instead of `nil`.

## Additional Middlewares

Using a regular `http.Handler` you can attach more middlewares to the router:

```golang
var srv = middleware.New()

func foo(next http.Handler) http.Handler { … }
func bar(next http.Handler) http.Handler { … }

func main() {
    srv.Use(foo)
    srv.Use(bar)
    srv.GET("/", func(w http.ResponseWriter, r *http.Request) { … })
    srv.ListenAndServe(":3000")
}
```

A regular `http.Handler` uses the following template:

```golang
func foobar(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        […]
        next.ServeHTTP(w, r)
    })
}
```

When a request is matched and processed, the chain of middlewares is executed in the same order in which they were attached to the srv. In the example above, the chain will result in the following function calls:

```
foo(
    bar(
        func(http.ResponseWriter, *http.Request)
    )
)
```

## System Logs

* Error logs are sent to `os.Stderr`
* Access logs are sent to `os.Stdout`
* Disable all logs using `srv.DiscardLogs()`
