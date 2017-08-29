### Go Middleware

Primitive middleware for web services written using the [Go programming language](https://golang.org/). It handles common HTTP methods, static files, untrusted directory listing, non-defined URLs and request timeouts. The project is based on [HTTP Router](https://github.com/julienschmidt/httprouter) by Julien Schmidt and adapted to my personal needs. The timeouts are based on the article [Complete Guide to Go net/http Timeouts](https://blog.cloudflare.com/the-complete-guide-to-golang-net-http-timeouts/) by CloudFlare.

The project was originally part of [Pastio](https://github.com/cixtor/pastio), a Pastebin-like web application developed and later decoupled as an independent package to allow other developers to import this functionality in their own projects.

If _"router.ReadTimeout"_ is omitted the server will always wait for the continuation of the data expected by _"Content-Length"_, this is not a good practice so it is recommended to defined a minimal timeout to prevent malicious attacks against the web service.

All the handlers follow the same standard as _"http.HandlerFunc"_ so you are free to add more handlers in the middle to improve the cache, attach SSL certificates, or change the format of the logs.

The router supports dynamic named parameters in the form of `/a/b/:id/:foobar` and making use of the [context](https://golang.org/pkg/context/) package the values for `id` and `foobar` are passed to the handler. Make sure that all the routes are defined in a cascade from longest to smallest to prevent conflicts and allow the server to execute the correct handler.

```go
package main

import "github.com/cixtor/middleware"

const PUBLIC_FOLDER = "assets"

func main() {
    var app Application

    router := middleware.New()

    router.Port = "9090"
    router.ReadTimeout = 5
    router.WriteTimeout = 10

    router.STATIC(PUBLIC_FOLDER, "/assets")

    router.POST("/save", app.Save)
    router.GET("/modes", app.Modes)
    router.GET("/raw/:unique", app.RawCode)
    router.GET("/", app.Index)

    router.ListenAndServe()
}
```

You can implement the graceful server shutdown process with this:

```
router.GET("/stop", func(w http.ResponseWriter, r *http.Request) {
    w.Write([]byte("Shutting down server...\n"))
    router.Shutdown()
})
```

--- EOF
