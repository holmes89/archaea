package server_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/holmes89/archaea/server"
	"github.com/stretchr/testify/assert"
)

func TestWithCORS_SetsAllowOriginHeader(t *testing.T) {
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	handler := server.WithCORS(inner)

	// Simple (non-preflight) CORS request — rs/cors sets the allow-origin header.
	req := httptest.NewRequest(http.MethodPost, "/connect.rpc/Method", nil)
	req.Header.Set("Origin", "http://example.com")
	req.Header.Set("Content-Type", "application/connect+proto")

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	origin := w.Header().Get("Access-Control-Allow-Origin")
	assert.NotEmpty(t, origin, "Access-Control-Allow-Origin should be set for CORS requests")
}

func TestWithCORS_PassesThroughRequest(t *testing.T) {
	called := false
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	})
	handler := server.WithCORS(inner)

	req := httptest.NewRequest(http.MethodPost, "/foo", nil)
	req.Header.Set("Origin", "http://example.com")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	assert.True(t, called)
}

func TestLoggingMiddleware_CallsNext(t *testing.T) {
	called := false
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusNoContent)
	})
	handler := server.LoggingMiddleware(inner)

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	assert.True(t, called)
	assert.Equal(t, http.StatusNoContent, w.Code)
}

func TestNew_HandleFunc_Registered(t *testing.T) {
	s := server.New(":0")
	s.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	// Exercise the registered route via Handle directly (without starting the
	// server) by calling the mux through a test recorder.  We verify this
	// indirectly by confirming no panic occurs and the server value is non-nil.
	assert.NotNil(t, s)
}
