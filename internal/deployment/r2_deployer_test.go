package deployment

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

type capturedRequest struct {
	method       string
	path         string
	contentType  string
	cacheControl string
	auth         string
	body         string
}

// newFakeR2 starts an S3-compatible endpoint that records the request and
// replies with the given status code
func newFakeR2(t *testing.T, status int) (*httptest.Server, *capturedRequest) {
	t.Helper()
	captured := &capturedRequest{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		*captured = capturedRequest{
			method:       r.Method,
			path:         r.URL.Path,
			contentType:  r.Header.Get("Content-Type"),
			cacheControl: r.Header.Get("Cache-Control"),
			auth:         r.Header.Get("Authorization"),
			body:         string(body),
		}
		w.WriteHeader(status)
	}))
	t.Cleanup(server.Close)
	return server, captured
}

func TestR2DeployerDeployData(t *testing.T) {
	t.Run("UploadsObjectWithHeaders", func(t *testing.T) {
		server, captured := newFakeR2(t, http.StatusOK)
		deployer := newR2Deployer(server.URL, "test_key_id", "test_secret", "test-bucket")

		data := `{"faction":"test"}`
		if err := deployer.DeployData(strings.NewReader(data), int64(len(data)), "travel_data.json"); err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}

		if captured.method != http.MethodPut {
			t.Errorf("Expected method PUT, got '%s'", captured.method)
		}
		if captured.path != "/test-bucket/travel_data.json" {
			t.Errorf("Expected path '/test-bucket/travel_data.json', got '%s'", captured.path)
		}
		if captured.contentType != "application/json" {
			t.Errorf("Expected Content-Type 'application/json', got '%s'", captured.contentType)
		}
		if captured.cacheControl != r2CacheControl {
			t.Errorf("Expected Cache-Control '%s', got '%s'", r2CacheControl, captured.cacheControl)
		}
		if !strings.HasPrefix(captured.auth, "AWS4-HMAC-SHA256 Credential=test_key_id/") {
			t.Errorf("Expected SigV4 Authorization for test_key_id, got '%s'", captured.auth)
		}
		if captured.body != data {
			t.Errorf("Expected body '%s', got '%s'", data, captured.body)
		}
	})

	t.Run("BuffersNonSeekableReader", func(t *testing.T) {
		server, captured := newFakeR2(t, http.StatusOK)
		deployer := newR2Deployer(server.URL, "test_key_id", "test_secret", "test-bucket")

		data := `{"faction":"test"}`
		reader := io.NopCloser(strings.NewReader(data)) // hides Seek
		if err := deployer.DeployData(reader, int64(len(data)), "travel_data.json"); err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}

		if captured.body != data {
			t.Errorf("Expected body '%s', got '%s'", data, captured.body)
		}
	})

	t.Run("UnknownExtensionUsesOctetStream", func(t *testing.T) {
		server, captured := newFakeR2(t, http.StatusOK)
		deployer := newR2Deployer(server.URL, "test_key_id", "test_secret", "test-bucket")

		if err := deployer.DeployData(strings.NewReader("x"), 1, "blob"); err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}

		if captured.contentType != "application/octet-stream" {
			t.Errorf("Expected Content-Type 'application/octet-stream', got '%s'", captured.contentType)
		}
	})

	t.Run("ReturnsErrorOnFailure", func(t *testing.T) {
		server, _ := newFakeR2(t, http.StatusForbidden)
		deployer := newR2Deployer(server.URL, "test_key_id", "test_secret", "test-bucket")

		err := deployer.DeployData(strings.NewReader("{}"), 2, "travel_data.json")
		if err == nil {
			t.Fatal("Expected error for 403 response, got nil")
		}
		if !strings.Contains(err.Error(), "test-bucket") {
			t.Errorf("Expected error message to contain bucket name, got '%s'", err.Error())
		}
	})
}

func TestNewR2Deployer(t *testing.T) {
	deployer := NewR2Deployer("abc123", "test_key_id", "test_secret", "test-bucket")

	endpoint := deployer.client.Options().BaseEndpoint
	if endpoint == nil || *endpoint != "https://abc123.r2.cloudflarestorage.com" {
		t.Errorf("Expected endpoint 'https://abc123.r2.cloudflarestorage.com', got %v", endpoint)
	}
	if deployer.bucket != "test-bucket" {
		t.Errorf("Expected bucket 'test-bucket', got '%s'", deployer.bucket)
	}
}
