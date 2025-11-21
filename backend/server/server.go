package server

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"path/filepath"
	"sync"
	"time"

	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
	"github.com/skarlso/kube_adventures/voting/backend/parser"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true // allow all origins since this is designed to either run with a reverse proxy or locally
	},
}

// Server manages the HTTP and WebSocket server.
type Server struct {
	mu              sync.RWMutex
	router          *mux.Router
	voteManager     *VoteManager
	storyEngine     *parser.StoryEngine
	currentNode     string
	history         []string // breadcrumb of visited chapter IDs
	staticFS        fs.FS
	presenterSecret string
}

// NewServer creates a new server instance with embedded filesystem.
func NewServer(storyPath, contentDir string, staticFS fs.FS, presenterSecret string) (*Server, error) {
	engine, err := parser.NewStoryEngine(storyPath, contentDir)
	if err != nil {
		return nil, fmt.Errorf("failed to create story engine: %w", err)
	}

	if errors := engine.ValidateStory(); len(errors) > 0 {
		log.Println("Story validation warnings:")

		for _, err := range errors {
			log.Printf("  - %v", err)
		}
	}

	s := &Server{
		router:          mux.NewRouter(),
		voteManager:     NewVoteManager(),
		storyEngine:     engine,
		currentNode:     engine.Story.Flow.Start,
		history:         []string{},
		staticFS:        staticFS,
		presenterSecret: presenterSecret,
	}

	s.setupRoutes()

	go s.voteManager.Run()

	return s, nil
}

func (s *Server) setupRoutes() {
	api := s.router.PathPrefix("/api").Subrouter()

	// no auth
	api.HandleFunc("/chapter/current", s.handleGetCurrentChapter).Methods("GET")
	api.HandleFunc("/chapter/{id}", s.handleGetChapter).Methods("GET")
	api.HandleFunc("/results/{questionId}", s.handleGetResults).Methods("GET")

	// with auth
	api.HandleFunc("/start-voting", s.requirePresenterAuth(s.handleStartVoting)).Methods("POST")
	api.HandleFunc("/advance", s.requirePresenterAuth(s.handleAdvance)).Methods("POST")
	api.HandleFunc("/restart", s.requirePresenterAuth(s.handleRestart)).Methods("POST")
	api.HandleFunc("/restart-voting", s.requirePresenterAuth(s.handleRestartVoting)).Methods("POST")
	api.HandleFunc("/go-back", s.requirePresenterAuth(s.handleGoBack)).Methods("POST")

	s.router.HandleFunc("/ws", s.handleWebSocket)

	fileServer := http.FileServer(http.FS(s.staticFS))
	s.router.PathPrefix("/presenter").Handler(s.requirePresenterAuthMiddleware(fileServer))
	s.router.PathPrefix("/").Handler(fileServer)
}

// requirePresenterAuth is a simple middleware for presenter authentication.
// Accepts both Bearer token and Basic Auth.
func (s *Server) requirePresenterAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// skip if there is no secret defined
		if s.presenterSecret == "" {
			next(w, r)

			return
		}

		_, password, ok := r.BasicAuth()
		if ok && password == s.presenterSecret {
			next(w, r)

			return
		}

		authHeader := r.Header.Get("Authorization")

		const prefix = "Bearer "
		if len(authHeader) >= len(prefix) && authHeader[:len(prefix)] == prefix {
			token := authHeader[len(prefix):]
			if token == s.presenterSecret {
				next(w, r)

				return
			}
		}

		w.Header().Set("WWW-Authenticate", `Basic realm="Presenter Access"`)
		http.Error(w, "unauthorized", http.StatusUnauthorized)
	}
}

// requirePresenterAuthMiddleware wraps an http.Handler with authentication.
// Uses HTTP Basic Auth for browser compatibility (triggers password popup).
func (s *Server) requirePresenterAuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// skip if there is no secret defined
		if s.presenterSecret == "" {
			next.ServeHTTP(w, r)

			return
		}

		_, password, ok := r.BasicAuth()
		if !ok || password != s.presenterSecret {
			// this will trigger the password prompt on the presenter screen
			w.Header().Set("WWW-Authenticate", `Basic realm="Presenter Access"`)
			http.Error(w, "unauthorized", http.StatusUnauthorized)

			return
		}

		next.ServeHTTP(w, r)
	})
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

	if err := json.NewEncoder(w).Encode(map[string]any{
		"id":       chapterID,
		"metadata": chapter.Metadata,
		"content":  chapter.Content,
	}); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)

		return
	}
}

// handleGetCurrentChapter returns the current chapter.
func (s *Server) handleGetCurrentChapter(w http.ResponseWriter, r *http.Request) {
	s.mu.RLock()
	currentNode := s.currentNode
	s.mu.RUnlock()

	chapter, err := s.storyEngine.GetChapter(currentNode)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)

		return
	}

	w.Header().Set("Content-Type", "application/json")

	if err := json.NewEncoder(w).Encode(map[string]any{
		"id":       currentNode,
		"metadata": chapter.Metadata,
		"content":  chapter.Content,
	}); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)

		return
	}
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

	s.mu.RLock()
	currentNode := s.currentNode
	s.mu.RUnlock()

	chapter, err := s.storyEngine.GetChapter(currentNode)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)

		return
	}

	duration := time.Duration(req.Duration) * time.Second

	s.voteManager.StartVotingWithChoices(req.QuestionID, req.Choices, chapter.Metadata.Choices, chapter.Metadata.Question, duration, func(results map[string]int, winner string) {
		log.Printf("Voting complete. Winner: %s, Results: %v", winner, results)
	})

	w.WriteHeader(http.StatusOK)

	if err := json.NewEncoder(w).Encode(map[string]any{
		"status": "voting_started",
	}); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)

		return
	}
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

	s.mu.Lock()
	defer s.mu.Unlock()

	s.history = append(s.history, s.currentNode)

	var (
		nextChapter *parser.Chapter
		err         error
	)

	if req.ChoiceID != "" {
		nextChapter, err = s.storyEngine.GetChapterByChoice(s.currentNode, req.ChoiceID)
	} else {
		nextChapter, err = s.storyEngine.GetNextChapter(s.currentNode)
	}

	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)

		return
	}

	s.currentNode = nextChapter.Metadata.ID
	s.voteManager.BroadcastMessage("chapter_changed", map[string]any{
		"id":          s.currentNode,
		"metadata":    nextChapter.Metadata,
		"content":     nextChapter.Content,
		"can_go_back": len(s.history) > 0,
	})

	w.Header().Set("Content-Type", "application/json")

	if err := json.NewEncoder(w).Encode(map[string]any{
		"id":          s.currentNode,
		"metadata":    nextChapter.Metadata,
		"content":     nextChapter.Content,
		"can_go_back": len(s.history) > 0,
	}); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)

		return
	}
}

// handleRestart restarts the entire story from the beginning.
func (s *Server) handleRestart(w http.ResponseWriter, r *http.Request) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.currentNode = s.storyEngine.Story.Flow.Start
	s.history = []string{}

	chapter, err := s.storyEngine.GetChapter(s.currentNode)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)

		return
	}

	// THIS IS IMPORTANT! Reset the voting state when the story restarts. This should also be done when going back.
	s.voteManager.ResetVoting()
	s.voteManager.BroadcastMessage("story_restarted", map[string]any{
		"id":       s.currentNode,
		"metadata": chapter.Metadata,
		"content":  chapter.Content,
	})

	w.Header().Set("Content-Type", "application/json")

	if err := json.NewEncoder(w).Encode(map[string]any{
		"id":       s.currentNode,
		"metadata": chapter.Metadata,
		"content":  chapter.Content,
	}); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)

		return
	}
}

// handleRestartVoting restarts the current voting session.
func (s *Server) handleRestartVoting(w http.ResponseWriter, r *http.Request) {
	s.mu.RLock()
	currentNode := s.currentNode
	s.mu.RUnlock()

	chapter, err := s.storyEngine.GetChapter(currentNode)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)

		return
	}

	if chapter.Metadata.Type != "decision" {
		http.Error(w, "current chapter is not a decision point", http.StatusBadRequest)

		return
	}

	s.voteManager.ResetVoting()

	w.Header().Set("Content-Type", "application/json")

	if err := json.NewEncoder(w).Encode(map[string]any{
		"status": "voting_reset",
	}); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)

		return
	}
}

// handleGoBack goes back to the previous chapter.
func (s *Server) handleGoBack(w http.ResponseWriter, r *http.Request) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if len(s.history) == 0 {
		http.Error(w, "no history to go back to", http.StatusBadRequest)

		return
	}

	currentChapterID := s.currentNode
	previousNode := s.history[len(s.history)-1]
	s.history = s.history[:len(s.history)-1]

	// prev chapter
	chapter, err := s.storyEngine.GetChapter(previousNode)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)

		return
	}

	s.currentNode = previousNode
	// clear for current question only
	s.voteManager.ClearQuestionVotes(currentChapterID)

	// inform all clients about the chapter change
	s.voteManager.BroadcastMessage("chapter_changed", map[string]any{
		"id":          s.currentNode,
		"metadata":    chapter.Metadata,
		"content":     chapter.Content,
		"can_go_back": len(s.history) > 0,
	})

	w.Header().Set("Content-Type", "application/json")

	if err := json.NewEncoder(w).Encode(map[string]any{
		"id":          s.currentNode,
		"metadata":    chapter.Metadata,
		"content":     chapter.Content,
		"can_go_back": len(s.history) > 0,
	}); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)

		return
	}
}

// handleGetResults returns voting results for a question.
func (s *Server) handleGetResults(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	questionID := vars["questionId"]

	results := s.voteManager.GetResults(questionID)

	w.Header().Set("Content-Type", "application/json")

	if err := json.NewEncoder(w).Encode(map[string]any{
		"question_id": questionID,
		"results":     results,
	}); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)

		return
	}
}

// handleWebSocket handles WebSocket connections.
func (s *Server) handleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("Failed to upgrade connection: %v", err)

		return
	}

	s.voteManager.RegisterClient(conn)

	// read messages from client
	go func() {
		defer func() {
			s.voteManager.UnregisterClient(conn)
			_ = conn.Close()
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

	server := http.Server{
		Addr:        addr,
		IdleTimeout: time.Minute,
		ReadTimeout: 10 * time.Second,
		Handler:     s.router,
	}

	return server.ListenAndServe()
}
