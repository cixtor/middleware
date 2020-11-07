// Package middleware is an HTTP middleware for web services.
//
//   * Follows the Go standards to comply with http.HandlerFunc
//   * Handles different HTTP methods (GET, POST, OPTIONS, etc)
//   * Handles IP address whitelisting to allow access to specific routes
//   * Handles IP address blacklisting to deny access to specific routes
//   * Handles serving of static files (CSS, JavaScript, Images, etc)
//   * Handles HTTP requests with failures related with timeouts
//   * Blocks directory listing of folders without an index file
//   * Handles HTTP requests to non-existing HTTP routes
//   * Supports dynamic named parameters in the URL
//
// Below is an example of a basic web server with this router:
//
//   var router = middleware.New()
//
//   func init() {
//       router.IdleTimeout = time.Second * 10
//       router.ReadTimeout = time.Second * 10
//       router.WriteTimeout = time.Second * 10
//       router.ShutdownTimeout = time.Second * 10
//       router.ReadHeaderTimeout = time.Second * 10
//
//       router.STATIC("/var/www/public_html", "/assets")
//
//       router.GET("/", index)
//   }
//
//   func main() {
//       shutdown := make(chan os.Signal, 1)
//       signal.Notify(shutdown, syscall.SIGINT, syscall.SIGTERM)
//       go func() { log.Fatal(router.ListenAndServe(":4000")) }()
//       <-shutdown
//       router.Shutdown()
//       log.Println("finished")
//   }
//
//   func index(w http.ResponseWriter, r *http.Request) {
//       _, _ = w.Write([]byte("Hello World!\n"))
//   }
package middleware
