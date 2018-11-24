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
//       router.IdleTimeout = 10 * time.Second
//       router.ReadTimeout = 10 * time.Second
//       router.WriteTimeout = 10 * time.Second
//       router.ShutdownTimeout = 10 * time.Second
//       router.ReadHeaderTimeout = 10 * time.Second
//
//       router.STATIC("/var/www/public_html", "/assets")
//
//       router.GET("/", index)
//   }
//
//   func main() {
//       shutdown := make(chan os.Signal, 1)
//       signal.Notify(shutdown, syscall.SIGINT, syscall.SIGTERM)
//       go func() {
//           <-shutdown
//           router.Shutdown()
//       }()
//       router.ListenAndServe()
//   }
//
//   func index(w http.ResponseWriter, r *http.Request) {
//       w.Write([]byte("Hello World!\n"))
//   }
package middleware
