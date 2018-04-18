### Go Middleware

Primitive middleware for web services written using the [Go programming language](https://golang.org/). It handles common HTTP methods, static files, untrusted directory listing, non-defined URLs and request timeouts. The project is based on [HTTP Router](https://github.com/julienschmidt/httprouter) by Julien Schmidt and adapted to my personal needs. The timeouts are based on the article [Complete Guide to Go net/http Timeouts](https://blog.cloudflare.com/the-complete-guide-to-golang-net-http-timeouts/) by CloudFlare.

If _"router.ReadTimeout"_ is omitted the server will always wait for the continuation of the data expected by _"Content-Length"_, this is not a good practice so it is recommended to defined a minimal timeout to prevent malicious attacks against the web service.

All the handlers follow the same standard as _"http.HandlerFunc"_ so you are free to add more handlers in the middle to improve the cache, attach SSL certificates, or change the format of the logs.

The router supports dynamic named parameters in the form of `/a/b/:id/:foobar` and making use of the [context](https://golang.org/pkg/context/) package the values for `id` and `foobar` are passed to the handler. Make sure that all the routes are defined in a cascade from longest to smallest to prevent conflicts and allow the server to execute the correct handler.

### Usage

```go
package main

import "github.com/cixtor/middleware"

func main() {
    var app Application

    router := middleware.New()

    router.Port = "8080"
    router.IdleTimeout = 5
    router.ReadTimeout = 5
    router.WriteTimeout = 10

    router.STATIC("/var/www/public_html", "/assets")

    router.POST("/save", app.Save)
    router.GET("/modes", app.Modes)
    router.GET("/raw/:unique", app.RawCode)
    router.GET("/", app.Index)

    router.ListenAndServe()
}
```

### TLS Support

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

### Graceful Shutdown

```go
func main() {
    router := middleware.New()

    shutdown := make(chan os.Signal, 1)
    signal.Notify(shutdown, os.Interrupt)

    go func() {
        <-shutdown
        router.Shutdown()
    }()

    router.ListenAndServe()
}
```

### System Logs

* Access logs are sent to `os.Stdout`
* Error logs are sent to `os.Stderr`
* Redirect using `webserver 1>access.log 2>errors.logs`
