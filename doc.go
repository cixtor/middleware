// Package middleware is an HTTP middleware for web services.
//
//   - Follows the Go standards to comply with http.HandlerFunc
//   - Handles different HTTP methods (GET, POST, OPTIONS, etc)
//   - Handles IP address whitelisting to allow access to specific routes
//   - Handles IP address blacklisting to deny access to specific routes
//   - Handles serving of static files (CSS, JavaScript, Images, etc)
//   - Handles HTTP requests with failures related with timeouts
//   - Blocks directory listing of folders without an index file
//   - Handles HTTP requests to non-existing HTTP routes
//   - Supports dynamic named parameters in the URL
//
// Example of a web server:
//
//	var srv = middleware.New()
//
//	func init() {
//	    srv.GET("/", index)
//	}
//
//	func main() {
//	    quit := make(chan os.Signal, 1)
//	    signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
//	    go func() { <-quit; /* close resources */ ; srv.Shutdown() }()
//
//	    srv.STATIC("/var/www/public_html", "/assets")
//
//	    if err := srv.ListenAndServe(":3000"); err != nil {
//	        log.Fatal(err)
//	    }
//	}
//
//	func index(w http.ResponseWriter, r *http.Request) {
//	    _, _ = w.Write([]byte("Hello World!\n"))
//	}
package middleware
