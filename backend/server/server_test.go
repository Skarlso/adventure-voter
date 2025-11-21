package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

// setupTestServer creates a test server with sample content
func setupTestServer(t *testing.T) (*Server, string) {
	t.Helper()

	tmpDir := t.TempDir()
	contentDir := filepath.Join(tmpDir, "chapters")
	staticDir := filepath.Join(tmpDir, "static")

	if err := os.Mkdir(contentDir, 0755); err != nil {
		t.Fatalf("failed to create content dir: %v", err)
	}
	if err := os.Mkdir(staticDir, 0755); err != nil {
		t.Fatalf("failed to create static dir: %v", err)
	}

	// Create simplified index file
	indexContent := `start: intro`

	indexFile := filepath.Join(tmpDir, "story.yaml")
	if err := os.WriteFile(indexFile, []byte(indexContent), 0600); err != nil {
		t.Fatalf("failed to create index file: %v", err)
	}

	// Create chapter files
	chapters := map[string]string{
		"intro.md": `---
id: intro
type: story
next: choice1
---
# Introduction
Welcome!`,
		"choice.md": `---
id: choice1
type: decision
timer: 60
question: Choose your path
choices:
  - id: opt-a
    label: Option A
    next: path-a
  - id: opt-b
    label: Option B
    next: path-b
---
# Choose your path`,
		"path-a.md": `---
id: path-a
type: story
---
# Path A`,
		"path-b.md": `---
id: path-b
type: game-over
---
# Game Over`,
	}

	for filename, content := range chapters {
		path := filepath.Join(contentDir, filename)
		if err := os.WriteFile(path, []byte(content), 0600); err != nil {
			t.Fatalf("failed to create %s: %v", filename, err)
		}
	}

	server, err := NewServer(indexFile, contentDir, staticDir, "")
	if err != nil {
		t.Fatalf("failed to create server: %v", err)
	}

	// Start vote manager
	go server.voteManager.Run()

	return server, tmpDir
}

func TestNewServer(t *testing.T) {
	server, tmpDir := setupTestServer(t)
	defer os.RemoveAll(tmpDir)

	if server.router == nil {
		t.Error("router should be initialized")
	}

	if server.voteManager == nil {
		t.Error("voteManager should be initialized")
	}

	if server.storyEngine == nil {
		t.Error("storyEngine should be initialized")
	}

	if server.currentNode != "intro" {
		t.Errorf("currentNode = %q, want %q", server.currentNode, "intro")
	}
}

func TestNewServer_InvalidPaths(t *testing.T) {
	tests := []struct {
		name       string
		storyPath  string
		contentDir string
	}{
		{"invalid story path", "/nonexistent/story.yaml", "/tmp"},
		{"invalid story yaml", "", "/tmp"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewServer(tt.storyPath, tt.contentDir, "/tmp", "")
			if err == nil {
				t.Error("expected error for invalid paths")
			}
		})
	}
}

func TestHandleGetCurrentChapter(t *testing.T) {
	server, tmpDir := setupTestServer(t)
	defer os.RemoveAll(tmpDir)

	req := httptest.NewRequest("GET", "/api/chapter/current", nil)
	w := httptest.NewRecorder()

	server.handleGetCurrentChapter(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}

	var response map[string]any
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if response["id"] != "intro" {
		t.Errorf("chapter id = %v, want %q", response["id"], "intro")
	}

	// Content should be present
	if _, ok := response["content"]; !ok {
		t.Error("response should contain content")
	}
}

func TestHandleGetChapter(t *testing.T) {
	server, tmpDir := setupTestServer(t)
	defer os.RemoveAll(tmpDir)

	tests := []struct {
		name       string
		chapterID  string
		wantStatus int
		wantID     string
	}{
		{
			name:       "valid chapter",
			chapterID:  "choice1",
			wantStatus: http.StatusOK,
			wantID:     "choice1",
		},
		{
			name:       "nonexistent chapter",
			chapterID:  "nonexistent",
			wantStatus: http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/api/chapter/"+tt.chapterID, nil)
			w := httptest.NewRecorder()

			// Manually set path variables for mux
			server.router.ServeHTTP(w, req)

			if w.Code != tt.wantStatus {
				t.Errorf("status = %d, want %d", w.Code, tt.wantStatus)
			}

			if tt.wantStatus == http.StatusOK {
				var response map[string]any
				if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
					t.Fatalf("failed to decode response: %v", err)
				}

				if response["id"] != tt.wantID {
					t.Errorf("chapter id = %v, want %q", response["id"], tt.wantID)
				}
			}
		})
	}
}

func TestHandleStartVoting(t *testing.T) {
	server, tmpDir := setupTestServer(t)
	defer os.RemoveAll(tmpDir)

	// First navigate to a decision point
	server.currentNode = "choice1"

	reqBody := map[string]any{
		"question_id": "choice1",
		"choices":     []string{"opt-a", "opt-b"},
		"duration":    5,
	}

	body, _ := json.Marshal(reqBody)
	req := httptest.NewRequest("POST", "/api/start-voting", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	server.handleStartVoting(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}

	if !server.voteManager.IsVotingActive() {
		t.Error("voting should be active")
	}

	// Verify response
	var response map[string]any
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if response["status"] != "voting_started" {
		t.Errorf("status = %v, want %q", response["status"], "voting_started")
	}
}

func TestHandleAdvance(t *testing.T) {
	server, tmpDir := setupTestServer(t)
	defer os.RemoveAll(tmpDir)

	tests := []struct {
		name           string
		currentNode    string
		choiceID       string
		wantNextID     string
		wantStatus     int
	}{
		{
			name:        "advance from story chapter",
			currentNode: "intro",
			choiceID:    "",
			wantNextID:  "choice1",
			wantStatus:  http.StatusOK,
		},
		{
			name:        "advance by choice",
			currentNode: "choice1",
			choiceID:    "opt-a",
			wantNextID:  "path-a",
			wantStatus:  http.StatusOK,
		},
		{
			name:        "invalid choice",
			currentNode: "choice1",
			choiceID:    "invalid",
			wantStatus:  http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server.currentNode = tt.currentNode
			server.history = []string{} // Reset history

			reqBody := map[string]any{
				"choice_id": tt.choiceID,
			}

			body, _ := json.Marshal(reqBody)
			req := httptest.NewRequest("POST", "/api/advance", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			server.handleAdvance(w, req)

			if w.Code != tt.wantStatus {
				t.Errorf("status = %d, want %d", w.Code, tt.wantStatus)
			}

			if tt.wantStatus == http.StatusOK {
				var response map[string]any
				if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
					t.Fatalf("failed to decode response: %v", err)
				}

				if response["id"] != tt.wantNextID {
					t.Errorf("next chapter id = %v, want %q", response["id"], tt.wantNextID)
				}

				// Verify history was updated
				if len(server.history) != 1 {
					t.Errorf("history length = %d, want 1", len(server.history))
				}
			}
		})
	}
}

func TestHandleRestart(t *testing.T) {
	server, tmpDir := setupTestServer(t)
	defer os.RemoveAll(tmpDir)

	// Navigate to a different chapter
	server.currentNode = "choice1"
	server.history = []string{"intro"}

	req := httptest.NewRequest("POST", "/api/restart", nil)
	w := httptest.NewRecorder()

	server.handleRestart(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}

	if server.currentNode != "intro" {
		t.Errorf("currentNode = %q, want %q", server.currentNode, "intro")
	}

	if len(server.history) != 0 {
		t.Errorf("history length = %d, want 0", len(server.history))
	}

	var response map[string]any
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if response["id"] != "intro" {
		t.Errorf("chapter id = %v, want %q", response["id"], "intro")
	}
}

func TestHandleGoBack(t *testing.T) {
	server, tmpDir := setupTestServer(t)
	defer os.RemoveAll(tmpDir)

	t.Run("successful go back", func(t *testing.T) {
		server.currentNode = "choice1"
		server.history = []string{"intro"}

		req := httptest.NewRequest("POST", "/api/go-back", nil)
		w := httptest.NewRecorder()

		server.handleGoBack(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
		}

		if server.currentNode != "intro" {
			t.Errorf("currentNode = %q, want %q", server.currentNode, "intro")
		}

		if len(server.history) != 0 {
			t.Errorf("history length = %d, want 0", len(server.history))
		}
	})

	t.Run("no history", func(t *testing.T) {
		server.currentNode = "intro"
		server.history = []string{}

		req := httptest.NewRequest("POST", "/api/go-back", nil)
		w := httptest.NewRecorder()

		server.handleGoBack(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
		}
	})
}

func TestHandleGetResults(t *testing.T) {
	server, tmpDir := setupTestServer(t)
	defer os.RemoveAll(tmpDir)

	// Start voting and submit some votes
	questionID := "test-question"
	server.voteManager.StartVoting(questionID, []string{"a", "b"}, 1*time.Second, nil)
	server.voteManager.SubmitVote("voter-1", "a")
	server.voteManager.SubmitVote("voter-2", "a")
	server.voteManager.SubmitVote("voter-3", "b")

	req := httptest.NewRequest("GET", "/api/results/"+questionID, nil)
	w := httptest.NewRecorder()

	server.router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}

	var response map[string]any
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	results, ok := response["results"].(map[string]any)
	if !ok {
		t.Fatal("results should be a map")
	}

	if int(results["a"].(float64)) != 2 {
		t.Errorf("votes for 'a' = %v, want 2", results["a"])
	}
}

func TestHandleRestartVoting(t *testing.T) {
	server, tmpDir := setupTestServer(t)
	defer os.RemoveAll(tmpDir)

	t.Run("restart voting at decision point", func(t *testing.T) {
		server.currentNode = "choice1"

		req := httptest.NewRequest("POST", "/api/restart-voting", nil)
		w := httptest.NewRecorder()

		server.handleRestartVoting(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
		}
	})

	t.Run("restart voting at non-decision point", func(t *testing.T) {
		server.currentNode = "intro"

		req := httptest.NewRequest("POST", "/api/restart-voting", nil)
		w := httptest.NewRecorder()

		server.handleRestartVoting(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
		}
	})
}

func TestWebSocketConnection(t *testing.T) {
	server, tmpDir := setupTestServer(t)
	defer os.RemoveAll(tmpDir)

	// Create test HTTP server
	ts := httptest.NewServer(server.router)
	defer ts.Close()

	// Convert http:// to ws://
	wsURL := "ws" + strings.TrimPrefix(ts.URL, "http") + "/ws"

	// Connect WebSocket
	ws, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("failed to connect websocket: %v", err)
	}
	defer ws.Close()

	// Should receive initial state message
	var msg Message
	if err := ws.ReadJSON(&msg); err != nil {
		t.Fatalf("failed to read state message: %v", err)
	}

	if msg.Type != "state" {
		t.Errorf("message type = %q, want %q", msg.Type, "state")
	}
}

func TestWebSocketVoting(t *testing.T) {
	server, tmpDir := setupTestServer(t)
	defer os.RemoveAll(tmpDir)

	ts := httptest.NewServer(server.router)
	defer ts.Close()

	wsURL := "ws" + strings.TrimPrefix(ts.URL, "http") + "/ws"

	ws, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("failed to connect websocket: %v", err)
	}
	defer ws.Close()

	// Read initial state
	var stateMsg Message
	ws.ReadJSON(&stateMsg)

	// Start voting
	server.voteManager.StartVoting("test-q", []string{"a", "b"}, 2*time.Second, nil)

	// Should receive voting_started message
	var startMsg Message
	if err := ws.ReadJSON(&startMsg); err != nil {
		t.Fatalf("failed to read voting_started message: %v", err)
	}

	if startMsg.Type != "voting_started" {
		t.Errorf("message type = %q, want %q", startMsg.Type, "voting_started")
	}

	// Submit vote via WebSocket
	voteMsg := VoteMessage{
		Type:     "vote",
		VoterID:  "ws-voter-1",
		ChoiceID: "a",
	}

	if err := ws.WriteJSON(voteMsg); err != nil {
		t.Fatalf("failed to send vote: %v", err)
	}

	// Should receive vote_update message
	var updateMsg Message
	if err := ws.ReadJSON(&updateMsg); err != nil {
		t.Fatalf("failed to read vote_update message: %v", err)
	}

	if updateMsg.Type != "vote_update" {
		t.Errorf("message type = %q, want %q", updateMsg.Type, "vote_update")
	}
}

func TestInvalidJSONRequests(t *testing.T) {
	server, tmpDir := setupTestServer(t)
	defer os.RemoveAll(tmpDir)

	tests := []struct {
		name     string
		path     string
		method   string
		body     string
		wantCode int
	}{
		{
			name:     "invalid json for start voting",
			path:     "/api/start-voting",
			method:   "POST",
			body:     "{invalid json}",
			wantCode: http.StatusBadRequest,
		},
		{
			name:     "invalid json for advance",
			path:     "/api/advance",
			method:   "POST",
			body:     "{invalid}",
			wantCode: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, tt.path, strings.NewReader(tt.body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			server.router.ServeHTTP(w, req)

			if w.Code != tt.wantCode {
				t.Errorf("status = %d, want %d", w.Code, tt.wantCode)
			}
		})
	}
}
