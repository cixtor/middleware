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
    "github.com/cixtor/middleware"
)

func main() {
    srv := middleware.New()
    srv.GET("/", index)
    srv.ListenAndServe(":3000")
}

func index(w http.ResponseWriter, r *http.Request) {
    _, _ = w.Write([]byte("Hello World!\n"))
}
```

## Server Timeouts

Override one or more of the (default) server timeouts:

```golang
srv.ReadTimeout       = time.Second * 2
srv.ReadHeaderTimeout = time.Second * 1
srv.WriteTimeout      = time.Second * 2
srv.IdleTimeout       = time.Second * 2
srv.ShutdownTimeout   = time.Millisecond * 100
```

Base your calculations on this HTTP request diagram:

```plain
┌──────────────────────────────────http.Request───────────────────────────────────┐
│ Accept                                                                          │
│ ┌──────┬───────────┬──────────────────────────┬──────────────────┬────────────┐ │
│ │      │    TLS    │         Request          │     Response     │            │ │
│ │ Wait │ Handshake ├───────────────────┬──────┼───────────┬──────┤    Idle    │ │
│ │      │           │      Headers      │ Body │  Headers  │ Body │            │ │
│ └──────┴───────────┴───────────────────┴──────┴───────────┴──────┴────────────┘ │
│                                        ├───────ServerHTTP────────┤              │
│                                                                    IdleTimeout  │
│                    ├─ReadHeaderTimeout─┤                         ├(keep-alive)┤ │
│                                                                                 │
│ ├──────────────────ReadTimeout────────────────┤                                 │
│                                                                                 │
│ ├ ─ ─ ─ ─WriteTimeout (TLS only) ─ ─ ─ ┼──────WriteTimeout──────┤               │
│                                                                                 │
│                                        ├───http.TimeoutHandler──┤               │
└─────────────────────────────────────────────────────────────────────────────────┘
```

## Serving Static Files

```golang
srv.STATIC("/var/www/public_html", "/assets")
```

In the example above, we are assumming that a directory located at `/var/www/public_html/` exists. With that in mind, every request to an URL with the `/assets/` prefix will be handled by the `http.ServeFiles()` method as long as the requested resource is pointing to an existing file.

A request to a directory returns "403 Forbidden" to prevent directory listing attacks.

A request to a nonexistent file returns "404 Not Found".

## Graceful Shutdown

You can implement a graceful shutdown with the following code:

```golang
func main() {
    quit := make(chan os.Signal, 1)
    signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
    go func() { <-quit; /* close resources */ ; srv.Shutdown() }()
    srv.ListenAndServe(":3000")
    fmt.Println("finished")
}
```

Common kill signals:

| Signal | Value | Effect | Notes |
|--------|-------|--------|-------|
| `SIGHUP` | 1 | Hangup ||
| `SIGINT` | 2 | Interrupt from keyboard ||
| `SIGKILL` | 9 | Kill signal | Cannot be caught, blocked or ignored |
| `SIGTERM` | 15 | Termination signal ||
| `SIGSTOP` | 17,19,23 | Stop the process | Cannot be caught, blocked or ignored |

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

func foo(next http.Handler) http.Handler { ... }
func bar(next http.Handler) http.Handler { ... }

func main() {
    srv.Use(foo)
    srv.Use(bar)
    srv.GET("/", func(w http.ResponseWriter, r *http.Request) { ... })
    srv.ListenAndServe(":3000")
}
```

A regular `http.Handler` uses the following template:

```golang
func foobar(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        [...]
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
* Implement the `middleware.Logger` interface to use your own logger
* Read `middleware.Logger` docs to implement request tracing (Prometheus)
