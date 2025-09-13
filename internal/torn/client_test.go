package torn

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestNewClient(t *testing.T) {
	client := NewClient("test_api_key")

	if client.apiKey != "test_api_key" {
		t.Errorf("Expected API key 'test_api_key', got '%s'", client.apiKey)
	}

	if client.client.Timeout != 30*time.Second {
		t.Errorf("Expected timeout 30s, got %v", client.client.Timeout)
	}

	if client.apiCallCount != 0 {
		t.Errorf("Expected API call count 0, got %d", client.apiCallCount)
	}
}

func TestAPICallCounter(t *testing.T) {
	client := NewClient("test_api_key")

	// Test initial count
	if count := client.GetAPICallCount(); count != 0 {
		t.Errorf("Expected initial count 0, got %d", count)
	}

	// Test increment
	client.IncrementAPICall()
	if count := client.GetAPICallCount(); count != 1 {
		t.Errorf("Expected count 1 after increment, got %d", count)
	}

	// Test multiple increments
	client.IncrementAPICall()
	client.IncrementAPICall()
	if count := client.GetAPICallCount(); count != 3 {
		t.Errorf("Expected count 3 after multiple increments, got %d", count)
	}

	// Test reset
	client.ResetAPICallCount()
	if count := client.GetAPICallCount(); count != 0 {
		t.Errorf("Expected count 0 after reset, got %d", count)
	}
}

// Note: Business logic tests have been moved to processor_test.go
// These tests focus on the infrastructure layer and backward compatibility

func TestMakeAPIRequest(t *testing.T) {
	// Create test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify request method and headers
		if r.Method != "GET" {
			t.Errorf("Expected GET request, got %s", r.Method)
		}

		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"test": "response"}`))
	}))
	defer server.Close()

	client := NewClient("test_api_key")
	ctx := context.Background()

	resp, err := client.makeAPIRequest(ctx, server.URL+"/test")
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	// Verify API call counter was incremented
	if count := client.GetAPICallCount(); count != 1 {
		t.Errorf("Expected API call count 1, got %d", count)
	}
}

func TestHandleAPIResponse(t *testing.T) {
	client := NewClient("test_api_key")

	t.Run("SuccessfulResponse", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"test": "data"}`))
		}))
		defer server.Close()

		resp, _ := http.Get(server.URL)
		defer resp.Body.Close()

		body, err := client.handleAPIResponse(resp)
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}

		expected := `{"test": "data"}`
		if string(body) != expected {
			t.Errorf("Expected body '%s', got '%s'", expected, string(body))
		}
	})

	t.Run("ErrorResponse", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(`{"error": {"msg": "Invalid API key"}}`))
		}))
		defer server.Close()

		resp, _ := http.Get(server.URL)
		defer resp.Body.Close()

		_, err := client.handleAPIResponse(resp)
		if err == nil {
			t.Fatal("Expected error for bad response, got nil")
		}

		if !strings.Contains(err.Error(), "Invalid API key") {
			t.Errorf("Expected error to contain 'Invalid API key', got: %s", err.Error())
		}
	})
}