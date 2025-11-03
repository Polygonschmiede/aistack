package services

import (
	"errors"
	"net"
	"net/http"
	"net/http/httptest"
	"syscall"
	"testing"
	"time"
)

func TestHealthCheck_Check_Success(t *testing.T) {
	// Create a test HTTP server
	server := startTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	hc := HealthCheck{
		URL:            server.URL,
		Timeout:        5 * time.Second,
		ExpectedStatus: http.StatusOK,
	}

	status, err := hc.Check()
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if status != HealthGreen {
		t.Errorf("Expected HealthGreen, got: %v", status)
	}
}

func TestHealthCheck_Check_WrongStatus(t *testing.T) {
	// Create a test HTTP server returning 500
	server := startTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	hc := HealthCheck{
		URL:            server.URL,
		Timeout:        5 * time.Second,
		ExpectedStatus: http.StatusOK,
	}

	status, err := hc.Check()
	if err == nil {
		t.Error("Expected error for wrong status code")
	}

	if status != HealthYellow {
		t.Errorf("Expected HealthYellow, got: %v", status)
	}
}

func TestHealthCheck_Check_Timeout(t *testing.T) {
	// Create a test HTTP server that delays response
	server := startTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(2 * time.Second)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	hc := HealthCheck{
		URL:            server.URL,
		Timeout:        100 * time.Millisecond, // Very short timeout
		ExpectedStatus: http.StatusOK,
	}

	status, err := hc.Check()
	if err == nil {
		t.Error("Expected timeout error")
	}

	if status != HealthRed {
		t.Errorf("Expected HealthRed on timeout, got: %v", status)
	}
}

func TestHealthCheck_CheckWithRetries(t *testing.T) {
	callCount := 0
	server := startTestServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		if callCount < 3 {
			w.WriteHeader(http.StatusInternalServerError)
		} else {
			w.WriteHeader(http.StatusOK)
		}
	}))
	defer server.Close()

	hc := HealthCheck{
		URL:            server.URL,
		Timeout:        5 * time.Second,
		ExpectedStatus: http.StatusOK,
	}

	status, err := hc.CheckWithRetries(5, 10*time.Millisecond)
	if err != nil {
		t.Fatalf("Expected success after retries, got error: %v", err)
	}

	if status != HealthGreen {
		t.Errorf("Expected HealthGreen after retries, got: %v", status)
	}

	if callCount != 3 {
		t.Errorf("Expected 3 calls, got: %d", callCount)
	}
}

func startTestServer(t *testing.T, handler http.Handler) *httptest.Server {
	t.Helper()

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		if errors.Is(err, syscall.EACCES) || errors.Is(err, syscall.EPERM) {
			t.Skipf("skipping HTTP server test: %v", err)
		}
		t.Fatalf("failed to create listener: %v", err)
	}

	server := httptest.NewUnstartedServer(handler)
	_ = server.Listener.Close()
	server.Listener = ln
	server.Start()
	return server
}
func TestDefaultHealthCheck(t *testing.T) {
	hc := DefaultHealthCheck("http://localhost:8080/health")

	if hc.URL != "http://localhost:8080/health" {
		t.Errorf("Expected URL to be set correctly")
	}

	if hc.Timeout != 10*time.Second {
		t.Errorf("Expected default timeout of 10s")
	}

	if hc.ExpectedStatus != http.StatusOK {
		t.Errorf("Expected default status of 200")
	}
}
