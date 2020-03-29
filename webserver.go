package middleware

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"net/http"
	"strings"
)

// startWebServer setups and starts the web server.
func (m *Middleware) startWebServer(f func() error) error {
	if err := m.validateHostname(m.Host); err != nil {
		return err
	}

	if m.ShutdownTimeout == 0 {
		m.ShutdownTimeout = defaultShutdownTimeout
	}

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

	go m.gracefulServerShutdown()

	m.Logger.Println("listening on", address)

	return f() /* ListenAndServe OR ListenAndServeTLS */
}

var ErrHostnameIsTooLong = errors.New("a valid hostname has a maximum of 253 ASCII characters")

var ErrInvalidHostnameLabel = errors.New("each hostname label must be between 1-63 characters long")

func (m *Middleware) validateHostname(s string) error {
	if len(s) > 253 {
		return ErrHostnameIsTooLong
	}

	p := strings.Split(s, ".")

	for i := 0; i < len(p); i++ {
		if p[i] == "" || len(p[i]) > 63 {
			return ErrInvalidHostnameLabel
		}
	}

	return nil
}

// ListenAndServe listens on the TCP network address srv.Addr
// and then calls server.Serve to handle requests on incoming
// connections. Accepted connections are configured to enable
// TCP keep-alives. If srv.Addr is blank, ":http" is used. The
// method always returns a non-nil error.
func (m *Middleware) ListenAndServe() error {
	return m.startWebServer(func() error {
		return m.serverInstance.ListenAndServe()
	})
}

// ListenAndServeTLS acts identically to ListenAndServe, except that it
// expects HTTPS connections. Additionally, files containing a certificate and
// matching private key for the server must be provided. If the certificate
// is signed by a certificate authority, the certFile should be the concatenation
// of the server's certificate, any intermediates, and the CA's certificate.
func (m *Middleware) ListenAndServeTLS(certFile string, keyFile string, cfg *tls.Config) error {
	return m.startWebServer(func() error {
		m.serverInstance.TLSConfig = cfg /* custom TLLS config */

		return m.serverInstance.ListenAndServeTLS(certFile, keyFile)
	})
}

// Shutdown stops the web server.
func (m *Middleware) Shutdown() {
	m.serverShutdown <- true
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
