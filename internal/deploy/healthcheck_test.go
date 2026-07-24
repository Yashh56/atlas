package deploy

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"
)

func TestHTTPHealthCheck_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	// Use the generic function. Note: for tests we'll need to mock the delay if we want it fast,
	// but success should be immediate.
	err := HTTPHealthCheck(ctx, server.URL, http.StatusOK)
	if err != nil {
		t.Fatalf("expected nil, got %v", err)
	}
}

func TestHTTPHealthCheck_RecoversFrom503(t *testing.T) {
	var attempts int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		current := atomic.AddInt32(&attempts, 1)
		if current < 2 {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	// In a real test, this would take ~2 seconds due to the sleep, which is acceptable for a single test.
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := HTTPHealthCheck(ctx, server.URL, http.StatusOK)
	if err != nil {
		t.Fatalf("expected recovery and nil error, got %v", err)
	}
	
	if atomic.LoadInt32(&attempts) != 2 {
		t.Errorf("expected exactly 2 attempts, got %d", attempts)
	}
}

func TestHTTPHealthCheck_PersistentFailure(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	// We don't want to wait 30s in test (5 retries with backoff), so we cancel early via context
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Millisecond)
	defer cancel()

	err := HTTPHealthCheck(ctx, server.URL, http.StatusOK)
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
}

func TestHTTPHealthCheck_404FailsImmediately(t *testing.T) {
	var attempts int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&attempts, 1)
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	err := HTTPHealthCheck(ctx, server.URL, http.StatusOK)
	if err == nil {
		t.Fatalf("expected error, got nil")
	}

	if atomic.LoadInt32(&attempts) != 1 {
		t.Errorf("expected exactly 1 attempt for 4xx, got %d", attempts)
	}
}
