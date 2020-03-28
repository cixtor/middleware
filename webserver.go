package middleware

import (
	"context"
	"crypto/tls"
	"fmt"
	"net/http"
)

// setDefaultSettings sets the default server settings.
func (m *Middleware) setDefaultSettings() {
	if m.Host == "" {
		m.Host = defaultHost
	}

	if m.ShutdownTimeout == 0 {
		m.ShutdownTimeout = defaultShutdownTimeout
	}
}

// startWebServer setups and starts the web server.
func (m *Middleware) startWebServer(f func()) {
	m.setDefaultSettings()

	address := fmt.Sprintf("%s:%d", m.Host, m.Port)
	m.serverShutdown = make(chan bool)
	m.serverInstance = &http.Server{
		Addr:              address,
		Handler:           m,
		IdleTimeout:       m.IdleTimeout,
		ReadTimeout:       m.ReadTimeout,
		WriteTimeout:      m.WriteTimeout,
		ReadHeaderTimeout: m.ReadHeaderTimeout,
	}

	go func() {
		m.Logger.Println("listening on", address)
		f() /* m.ListenAndServe OR m.ListenAndServeTLS */
	}()

	m.gracefulServerShutdown()
}

// gracefulServerShutdown shutdowns the server.
func (m *Middleware) gracefulServerShutdown() {
	<-m.serverShutdown /* wait shutdown */

	ctx, cancel := context.WithTimeout(
		context.Background(),
		m.ShutdownTimeout,
	)

	defer cancel()

	if err := m.serverInstance.Shutdown(ctx); err != nil {
		m.Logger.Println("sigint;", err)
		return
	}

	m.Logger.Println("server shutdown")
}

// ListenAndServe listens on the TCP network address srv.Addr
// and then calls server.Serve to handle requests on incoming
// connections. Accepted connections are configured to enable
// TCP keep-alives. If srv.Addr is blank, ":http" is used. The
// method always returns a non-nil error.
func (m *Middleware) ListenAndServe() {
	m.startWebServer(func() {
		err := m.serverInstance.ListenAndServe()

		if err != nil {
			m.Logger.Fatal(err)
		}
	})
}

// ListenAndServeTLS acts identically to ListenAndServe, except that it
// expects HTTPS connections. Additionally, files containing a certificate and
// matching private key for the server must be provided. If the certificate
// is signed by a certificate authority, the certFile should be the concatenation
// of the server's certificate, any intermediates, and the CA's certificate.
func (m *Middleware) ListenAndServeTLS(certFile string, keyFile string, cfg *tls.Config) {
	m.startWebServer(func() {
		m.serverInstance.TLSConfig = cfg /* custom TLLS config */

		err := m.serverInstance.ListenAndServeTLS(certFile, keyFile)

		if err != nil {
			m.Logger.Fatal(err)
		}
	})
}

// Shutdown stops the web server.
func (m *Middleware) Shutdown() {
	m.serverShutdown <- true
}
