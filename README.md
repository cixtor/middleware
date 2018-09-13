# Go HTTP Middleware [![GoReport](https://goreportcard.com/badge/github.com/cixtor/middleware)](https://goreportcard.com/report/github.com/cixtor/middleware) [![GoDoc](https://godoc.org/github.com/cixtor/middleware?status.svg)](https://godoc.org/github.com/cixtor/middleware)

HTTP middleware for web services [written in Go](https://golang.org/) _(aka. Golang)_.

* Follows the Go standards to comply with [http.HandlerFunc](https://golang.org/pkg/net/http/#HandlerFunc),
* Handles different HTTP methods _(GET, POST, OPTIONS, etc)_,
* Handles IP address whitelisting to allow access to specific routes,
* Handles IP address blacklisting to deny access to specific routes,
* Handles serving of static files _(CSS, JavaScript, Images, etc)_,
* Handles HTTP requests with failures related with timeouts,
* Blocks directory listing of folders without an index file,
* Handles HTTP requests to non-existing HTTP routes,
* Supports dynamic named parameters in the URL.

## Installation

```sh
go get -u github.com/cixtor/middleware
```

## Usage

Below you can find an example of how to implement a web server with this router:

```golang
package main

import "github.com/cixtor/middleware"

var router = middleware.New()

func init() {
    router.Port = "3000"
    router.IdleTimeout = 10
    router.ReadTimeout = 10
    router.WriteTimeout = 10
    router.ShutdownTimeout = 10
    router.ReadHeaderTimeout = 10

    router.STATIC("/var/www/public_html", "/assets")
}

func main() {
    router.ListenAndServe()
}
```

## http.HandlerFunc

The handler uses the Go [http.HandlerFunc](https://golang.org/pkg/net/http/#HandlerFunc) standard as you can see below:

```golang
package main

import "net/http"

func init() {
    router.GET("/", index)
}

func index(w http.ResponseWriter, r *http.Request) {
    w.Write([]byte("Hello World!\n"))
}
```

## Graceful Shutdown

A graceful shutdown can be added with the following code:

```golang
import (
    "os"
    "os/signal"
    "syscall"
}

func main() {
    shutdown := make(chan os.Signal, 1)
    signal.Notify(shutdown, syscall.SIGINT, syscall.SIGTERM)

    go func() {
        <-shutdown
        router.Shutdown()
    }()

    router.ListenAndServe()
}
```

## TLS Support

Generate the SSL certificates:

```
openssl genrsa -out server.key 2048

openssl ecparam -genkey -name secp384r1 -out server.key
# Country Name (2 letter code) []:CA
# State or Province Name (full name) []:British Columbia
# Locality Name (eg, city) []:Vancouver
# Organization Name (eg, company) []:Foobar Inc.
# Organizational Unit Name (eg, section) []:
# Common Name (eg, fully qualified host name) []:middleware.test
# Email Address []:webmaster@middleware.test

echo -e "127.0.0.1\tmiddleware.test" | sudo tee -a /etc/hosts
```

Use `router.ListenAndServeTLS("server.crt", "server.key", nil)` to start the web server.

Test the connection using cURL `curl --cacert server.crt "https://middleware.test:8080"`

Add a custom TLS configuration by passing a `&tls.Config{}` as the last parameter instead of `nil`.

## System Logs

* Access logs are sent to `os.Stdout`
* Error logs are sent to `os.Stderr`
* Redirect using `webserver 1>access.log 2>errors.logs`
