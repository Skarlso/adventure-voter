package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

func TestPresenterAuth(t *testing.T) {
	server, tmpDir := setupTestServer(t)
	defer os.RemoveAll(tmpDir)

	server.presenterSecret = "test-secret-123"

	tests := []struct {
		name           string
		endpoint       string
		method         string
		authHeader     string
		wantStatusCode int
	}{
		{
			name:           "no auth header",
			endpoint:       "/api/advance",
			method:         "POST",
			authHeader:     "",
			wantStatusCode: http.StatusUnauthorized,
		},
		{
			name:           "invalid auth format",
			endpoint:       "/api/advance",
			method:         "POST",
			authHeader:     "InvalidFormat",
			wantStatusCode: http.StatusUnauthorized,
		},
		{
			name:           "wrong secret",
			endpoint:       "/api/advance",
			method:         "POST",
			authHeader:     "Bearer wrong-secret",
			wantStatusCode: http.StatusUnauthorized,
		},
		{
			name:           "correct secret - advance",
			endpoint:       "/api/advance",
			method:         "POST",
			authHeader:     "Bearer test-secret-123",
			wantStatusCode: http.StatusOK, // Empty JSON body is valid for advance
		},
		{
			name:           "correct secret - start-voting",
			endpoint:       "/api/start-voting",
			method:         "POST",
			authHeader:     "Bearer test-secret-123",
			wantStatusCode: http.StatusOK, // Auth passed (will work with empty JSON)
		},
		{
			name:           "restart with auth",
			endpoint:       "/api/restart",
			method:         "POST",
			authHeader:     "Bearer test-secret-123",
			wantStatusCode: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body := bytes.NewBufferString("{}")
			req := httptest.NewRequest(tt.method, tt.endpoint, body)
			req.Header.Set("Content-Type", "application/json")

			if tt.authHeader != "" {
				req.Header.Set("Authorization", tt.authHeader)
			}

			w := httptest.NewRecorder()
			server.router.ServeHTTP(w, req)

			if w.Code != tt.wantStatusCode {
				t.Errorf("status = %d, want %d", w.Code, tt.wantStatusCode)
			}
		})
	}
}

func TestPresenterAuthDisabled(t *testing.T) {
	// Setup server WITHOUT auth (empty secret)
	server, tmpDir := setupTestServer(t)
	defer os.RemoveAll(tmpDir)

	// presenterSecret is "" by default, auth should be disabled

	// Should work without auth header
	req := httptest.NewRequest("POST", "/api/restart", bytes.NewBufferString("{}"))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	server.router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d (auth should be disabled)", w.Code, http.StatusOK)
	}
}

func TestPublicEndpointsNoAuth(t *testing.T) {
	server, tmpDir := setupTestServer(t)
	defer os.RemoveAll(tmpDir)

	server.presenterSecret = "test-secret-123"

	publicEndpoints := []struct {
		endpoint string
		method   string
	}{
		{"/api/chapter/current", "GET"},
		{"/api/chapter/intro", "GET"},
		{"/api/results/test-question", "GET"},
	}

	for _, ep := range publicEndpoints {
		t.Run(ep.endpoint, func(t *testing.T) {
			req := httptest.NewRequest(ep.method, ep.endpoint, nil)
			w := httptest.NewRecorder()

			server.router.ServeHTTP(w, req)

			// Should NOT be 401 Unauthorized
			if w.Code == http.StatusUnauthorized {
				t.Errorf("public endpoint %s returned 401, should be accessible without auth", ep.endpoint)
			}
		})
	}
}

func TestPresenterAuthIntegration(t *testing.T) {
	server, tmpDir := setupTestServer(t)
	defer os.RemoveAll(tmpDir)

	server.presenterSecret = "my-presentation-password"

	// 1. Advance to next chapter (requires auth)
	t.Run("advance with valid auth", func(t *testing.T) {
		body := map[string]string{"choice_id": ""}
		bodyJSON, _ := json.Marshal(body)

		req := httptest.NewRequest("POST", "/api/advance", bytes.NewBuffer(bodyJSON))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer my-presentation-password")

		w := httptest.NewRecorder()
		server.router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("advance failed with valid auth: status = %d", w.Code)
		}
	})

	// 2. Try to advance without auth (should fail)
	t.Run("advance without auth", func(t *testing.T) {
		body := map[string]string{"choice_id": ""}
		bodyJSON, _ := json.Marshal(body)

		req := httptest.NewRequest("POST", "/api/advance", bytes.NewBuffer(bodyJSON))
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		server.router.ServeHTTP(w, req)

		if w.Code != http.StatusUnauthorized {
			t.Errorf("advance without auth should return 401, got %d", w.Code)
		}
	})

	// 3. Get current chapter (public, no auth needed)
	t.Run("get current chapter without auth", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/chapter/current", nil)

		w := httptest.NewRecorder()
		server.router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("public endpoint failed: status = %d", w.Code)
		}
	})
}
