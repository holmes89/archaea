// Package server provides a minimal HTTP/2 API server with Connect-RPC CORS
// support, request logging, and graceful shutdown on SIGINT/SIGTERM.
package server

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	connectcors "connectrpc.com/cors"
	"github.com/rs/cors"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
)

// Server is a simple HTTP/2 server suitable for Connect-RPC APIs.
type Server struct {
	mux  *http.ServeMux
	addr string
	wrap []func(http.Handler) http.Handler
}

// New creates a Server that will listen on addr. Optional wrap functions are
// applied to the handler in the order given (outermost wraps first). Use this
// to inject OTel or other middleware without importing them in this package:
//
//	server.New(":9000", func(h http.Handler) http.Handler {
//	    return otelhttp.NewHandler(h, "my-service")
//	})
func New(addr string, wrap ...func(http.Handler) http.Handler) *Server {
	return &Server{
		mux:  http.NewServeMux(),
		addr: addr,
		wrap: wrap,
	}
}

// Handle registers handler for the given pattern, wrapped with Connect-RPC
// compatible CORS headers.
func (s *Server) Handle(pattern string, handler http.Handler) {
	s.mux.Handle(pattern, WithCORS(handler))
}

// HandleFunc registers fn for the given pattern. Unlike Handle, no CORS
// wrapping is applied — useful for /health or /debug endpoints.
func (s *Server) HandleFunc(pattern string, fn http.HandlerFunc) {
	s.mux.HandleFunc(pattern, fn)
}

// Run starts the HTTP/2 server and blocks until SIGINT, SIGTERM, or ctx is
// cancelled. Returns the signal or context error that caused shutdown.
func (s *Server) Run(ctx context.Context) error {
	var h http.Handler = LoggingMiddleware(s.mux)
	for _, w := range s.wrap {
		h = w(h)
	}

	srv := &http.Server{
		Addr:    s.addr,
		Handler: h2c.NewHandler(h, &http2.Server{}),
	}

	errs := make(chan error, 2)
	go func() {
		log.Printf("server: listening on %s", s.addr)
		errs <- srv.ListenAndServe()
	}()
	go func() {
		c := make(chan os.Signal, 1)
		signal.Notify(c, syscall.SIGINT, syscall.SIGTERM)
		defer signal.Stop(c)
		select {
		case sig := <-c:
			errs <- fmt.Errorf("signal: %s", sig)
		case <-ctx.Done():
			errs <- ctx.Err()
		}
	}()

	err := <-errs
	_ = srv.Shutdown(context.Background())
	return err
}

// WithCORS wraps h with Connect-RPC compatible CORS headers, allowing all
// origins. Suitable for personal self-hosted services.
func WithCORS(h http.Handler) http.Handler {
	c := cors.New(cors.Options{
		AllowedOrigins: []string{"*"},
		AllowedMethods: connectcors.AllowedMethods(),
		AllowedHeaders: connectcors.AllowedHeaders(),
		ExposedHeaders: connectcors.ExposedHeaders(),
	})
	return c.Handler(h)
}

// LoggingMiddleware logs the HTTP method and path of each request using the
// stdlib logger.
func LoggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Printf("server: %s %s", r.Method, r.URL.Path)
		next.ServeHTTP(w, r)
	})
}
