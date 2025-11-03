package server

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"path/filepath"
	"time"

	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
	"github.com/skarlso/kube_adventures/voting/backend/parser"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true // Allow all origins for development
	},
}

// Server manages the HTTP and WebSocket server.
type Server struct {
	router      *mux.Router
	voteManager *VoteManager
	storyEngine *parser.StoryEngine
	currentNode string
	history     []string // Navigation history for going back
	staticDir   string
}

// NewServer creates a new server instance.
func NewServer(storyPath, contentDir, staticDir string) (*Server, error) {
	engine, err := parser.NewStoryEngine(storyPath, contentDir)
	if err != nil {
		return nil, fmt.Errorf("failed to create story engine: %w", err)
	}

	// Validate story
	if errors := engine.ValidateStory(); len(errors) > 0 {
		log.Println("Story validation warnings:")

		for _, err := range errors {
			log.Printf("  - %v", err)
		}
	}

	s := &Server{
		router:      mux.NewRouter(),
		voteManager: NewVoteManager(),
		storyEngine: engine,
		currentNode: engine.Story.Flow.Start,
		history:     []string{},
		staticDir:   staticDir,
	}

	s.setupRoutes()

	// Start vote manager
	go s.voteManager.Run()

	return s, nil
}

func (s *Server) setupRoutes() {
	// API routes
	api := s.router.PathPrefix("/api").Subrouter()
	// More specific routes must come first
	api.HandleFunc("/chapter/current", s.handleGetCurrentChapter).Methods("GET")
	api.HandleFunc("/chapter/{id}", s.handleGetChapter).Methods("GET")
	api.HandleFunc("/start-voting", s.handleStartVoting).Methods("POST")
	api.HandleFunc("/advance", s.handleAdvance).Methods("POST")
	api.HandleFunc("/restart", s.handleRestart).Methods("POST")
	api.HandleFunc("/restart-voting", s.handleRestartVoting).Methods("POST")
	api.HandleFunc("/go-back", s.handleGoBack).Methods("POST")
	api.HandleFunc("/results/{questionId}", s.handleGetResults).Methods("GET")

	// WebSocket
	s.router.HandleFunc("/ws", s.handleWebSocket)

	// Static files
	s.router.PathPrefix("/").Handler(http.FileServer(http.Dir(s.staticDir)))
}

// handleGetChapter returns a specific chapter by ID.
func (s *Server) handleGetChapter(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	chapterID := vars["id"]

	chapter, err := s.storyEngine.GetChapter(chapterID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)

		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"id":       chapterID,
		"metadata": chapter.Metadata,
		"content":  chapter.Content,
	})
}

// handleGetCurrentChapter returns the current chapter.
func (s *Server) handleGetCurrentChapter(w http.ResponseWriter, r *http.Request) {
	chapter, err := s.storyEngine.GetChapter(s.currentNode)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)

		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"id":       s.currentNode,
		"metadata": chapter.Metadata,
		"content":  chapter.Content,
	})
}

// handleStartVoting starts a new voting session.
func (s *Server) handleStartVoting(w http.ResponseWriter, r *http.Request) {
	var req struct {
		QuestionID string   `json:"question_id"`
		Choices    []string `json:"choices"`
		Duration   int      `json:"duration"` // seconds
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)

		return
	}

	// Get the current chapter to access full choice metadata
	chapter, err := s.storyEngine.GetChapter(s.currentNode)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)

		return
	}

	duration := time.Duration(req.Duration) * time.Second

	// Pass the full choice objects to StartVoting
	s.voteManager.StartVotingWithChoices(req.QuestionID, req.Choices, chapter.Metadata.Choices, duration, func(results map[string]int, winner string) {
		log.Printf("Voting complete. Winner: %s, Results: %v", winner, results)
	})

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]any{
		"status": "voting_started",
	})
}

// handleAdvance advances to the next chapter based on choice.
func (s *Server) handleAdvance(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ChoiceID string `json:"choice_id"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)

		return
	}

	// Save current node to history before advancing
	s.history = append(s.history, s.currentNode)

	var (
		nextChapter *parser.Chapter
		err         error
	)

	if req.ChoiceID != "" {
		// Advance based on choice
		nextChapter, err = s.storyEngine.GetChapterByChoice(s.currentNode, req.ChoiceID)
	} else {
		// Advance to next chapter (for non-decision nodes)
		nextChapter, err = s.storyEngine.GetNextChapter(s.currentNode)
	}

	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)

		return
	}

	// Update current node
	s.currentNode = nextChapter.Metadata.ID

	// Broadcast chapter change to all clients
	s.voteManager.BroadcastMessage("chapter_changed", map[string]any{
		"id":          s.currentNode,
		"metadata":    nextChapter.Metadata,
		"content":     nextChapter.Content,
		"can_go_back": len(s.history) > 0,
	})

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"id":          s.currentNode,
		"metadata":    nextChapter.Metadata,
		"content":     nextChapter.Content,
		"can_go_back": len(s.history) > 0,
	})
}

// handleRestart restarts the entire story from the beginning.
func (s *Server) handleRestart(w http.ResponseWriter, r *http.Request) {
	// Reset to start
	s.currentNode = s.storyEngine.Story.Flow.Start
	s.history = []string{}

	// Get start chapter
	chapter, err := s.storyEngine.GetChapter(s.currentNode)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)

		return
	}

	// Reset voting
	s.voteManager.ResetVoting()

	// Broadcast restart to all clients
	s.voteManager.BroadcastMessage("story_restarted", map[string]any{
		"id":       s.currentNode,
		"metadata": chapter.Metadata,
		"content":  chapter.Content,
	})

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"id":       s.currentNode,
		"metadata": chapter.Metadata,
		"content":  chapter.Content,
	})
}

// handleRestartVoting restarts the current voting session.
func (s *Server) handleRestartVoting(w http.ResponseWriter, r *http.Request) {
	// Get current chapter
	chapter, err := s.storyEngine.GetChapter(s.currentNode)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)

		return
	}

	// Check if it's a decision point
	if chapter.Metadata.Type != "decision" {
		http.Error(w, "current chapter is not a decision point", http.StatusBadRequest)

		return
	}

	// Reset votes for this question
	s.voteManager.ResetVoting()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"status": "voting_reset",
	})
}

// handleGoBack goes back to the previous chapter.
func (s *Server) handleGoBack(w http.ResponseWriter, r *http.Request) {
	if len(s.history) == 0 {
		http.Error(w, "no history to go back to", http.StatusBadRequest)

		return
	}

	// Get current chapter ID before going back (to clear its votes)
	currentChapterID := s.currentNode

	// Pop from history
	previousNode := s.history[len(s.history)-1]
	s.history = s.history[:len(s.history)-1]

	// Get previous chapter
	chapter, err := s.storyEngine.GetChapter(previousNode)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)

		return
	}

	// Update current node
	s.currentNode = previousNode

	// Clear votes for the current question only
	s.voteManager.ClearQuestionVotes(currentChapterID)

	// Broadcast chapter change to all clients
	s.voteManager.BroadcastMessage("chapter_changed", map[string]any{
		"id":          s.currentNode,
		"metadata":    chapter.Metadata,
		"content":     chapter.Content,
		"can_go_back": len(s.history) > 0,
	})

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"id":          s.currentNode,
		"metadata":    chapter.Metadata,
		"content":     chapter.Content,
		"can_go_back": len(s.history) > 0,
	})
}

// handleGetResults returns voting results for a question.
func (s *Server) handleGetResults(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	questionID := vars["questionId"]

	results := s.voteManager.GetResults(questionID)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"question_id": questionID,
		"results":     results,
	})
}

// handleWebSocket handles WebSocket connections.
func (s *Server) handleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("Failed to upgrade connection: %v", err)

		return
	}

	s.voteManager.RegisterClient(conn)

	// Read messages from client
	go func() {
		defer func() {
			s.voteManager.UnregisterClient(conn)
			conn.Close()
		}()

		for {
			_, message, err := conn.ReadMessage()
			if err != nil {
				if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
					log.Printf("WebSocket error: %v", err)
				}

				break
			}

			if err := s.voteManager.HandleVoteMessage(message); err != nil {
				log.Printf("Error handling vote message: %v", err)
			}
		}
	}()
}

// Start starts the HTTP server.
func (s *Server) Start(addr string) error {
	log.Printf("Starting server on %s", addr)
	log.Printf("Content directory: %s", filepath.Dir(s.storyEngine.ContentDir))
	log.Printf("Static directory: %s", s.staticDir)

	return http.ListenAndServe(addr, s.router)
}
