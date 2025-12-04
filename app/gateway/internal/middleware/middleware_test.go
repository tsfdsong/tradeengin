package middleware

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRateLimitMiddleware_Handle_NilLimiter(t *testing.T) {
	middleware := NewRateLimitMiddleware(nil)

	called := false
	handler := middleware.Handle(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest("GET", "/test", nil)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if !called {
		t.Error("Expected handler to be called when limiter is nil")
	}
	if rr.Code != http.StatusOK {
		t.Errorf("Expected status OK, got %d", rr.Code)
	}
}

func TestRateLimitError(t *testing.T) {
	err := NewRateLimitError()

	if err.Code != 429 {
		t.Errorf("Expected code 429, got %d", err.Code)
	}

	if err.Error() != "Too many requests, please try again later" {
		t.Errorf("Unexpected error message: %s", err.Error())
	}

	// Test JSON serialization
	data, _ := json.Marshal(err)
	var decoded RateLimitError
	json.Unmarshal(data, &decoded)

	if decoded.Code != 429 {
		t.Errorf("JSON decode failed, expected code 429, got %d", decoded.Code)
	}
}

func TestBreakerMiddleware_Handle_NilBreaker(t *testing.T) {
	middleware := NewBreakerMiddleware(nil)

	called := false
	handler := middleware.Handle(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest("GET", "/test", nil)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if !called {
		t.Error("Expected handler to be called when breaker is nil")
	}
	if rr.Code != http.StatusOK {
		t.Errorf("Expected status OK, got %d", rr.Code)
	}
}

func TestCircuitBreakerError(t *testing.T) {
	err := NewCircuitBreakerError()

	if err.Code != 503 {
		t.Errorf("Expected code 503, got %d", err.Code)
	}

	if err.Error() != "Service temporarily unavailable, please try again later" {
		t.Errorf("Unexpected error message: %s", err.Error())
	}

	// Test JSON serialization
	data, _ := json.Marshal(err)
	var decoded CircuitBreakerError
	json.Unmarshal(data, &decoded)

	if decoded.Code != 503 {
		t.Errorf("JSON decode failed, expected code 503, got %d", decoded.Code)
	}
}

func TestResponseWriter(t *testing.T) {
	rr := httptest.NewRecorder()
	rw := &responseWriter{ResponseWriter: rr, statusCode: http.StatusOK}

	// Test WriteHeader
	rw.WriteHeader(http.StatusInternalServerError)

	if rw.statusCode != http.StatusInternalServerError {
		t.Errorf("Expected status 500, got %d", rw.statusCode)
	}
}
