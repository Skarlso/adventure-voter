package server

import (
	"encoding/json"
	"log"
	"maps"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/skarlso/kube_adventures/voting/backend/parser"
)

// VoteManager handles vote aggregation and broadcasting.
type VoteManager struct {
	mu              sync.RWMutex
	currentQuestion string
	votes           map[string]map[string]int // questionID -> choiceID -> count
	voters          map[string]string         // voterID -> choiceID (for current question)
	clients         map[*websocket.Conn]bool
	broadcast       chan *Message
	register        chan *websocket.Conn
	unregister      chan *websocket.Conn
	timer           *time.Timer
	timerDuration   time.Duration
	votingActive    bool
	onVoteComplete  func(results map[string]int, winner string)
}

// Message represents a WebSocket message.
type Message struct {
	Type    string         `json:"type"` // vote, results, state, timer, etc.
	Payload map[string]any `json:"payload"`
}

// NewVoteManager creates a new vote manager.
func NewVoteManager() *VoteManager {
	return &VoteManager{
		votes:      make(map[string]map[string]int),
		voters:     make(map[string]string),
		clients:    make(map[*websocket.Conn]bool),
		broadcast:  make(chan *Message, 256),
		register:   make(chan *websocket.Conn),
		unregister: make(chan *websocket.Conn),
	}
}

// Run starts the vote manager.
func (vm *VoteManager) Run() {
	for {
		select {
		case client := <-vm.register:
			vm.mu.Lock()
			vm.clients[client] = true
			vm.mu.Unlock()

			vm.sendState(client)

		case client := <-vm.unregister:
			vm.mu.Lock()

			if _, ok := vm.clients[client]; ok {
				delete(vm.clients, client)
				_ = client.Close()
			}

			vm.mu.Unlock()

		case message := <-vm.broadcast:
			vm.mu.RLock()

			clients := make([]*websocket.Conn, 0, len(vm.clients))
			for client := range vm.clients {
				clients = append(clients, client)
			}

			vm.mu.RUnlock()

			for _, client := range clients {
				err := client.WriteJSON(message)
				if err != nil {
					log.Printf("Error broadcasting to client: %v", err)

					vm.unregister <- client
				}
			}
		}
	}
}

// StartVoting begins a new voting session.
func (vm *VoteManager) StartVoting(questionID string, choices []string, duration time.Duration, onComplete func(map[string]int, string)) {
	vm.StartVotingWithChoices(questionID, choices, nil, "", duration, onComplete)
}

// StartVotingWithChoices begins a new voting session with full choice metadata.
func (vm *VoteManager) StartVotingWithChoices(questionID string, choiceIDs []string, choiceObjects []parser.Choice, question string, duration time.Duration, onComplete func(map[string]int, string)) {
	vm.mu.Lock()
	defer vm.mu.Unlock()

	// reset state
	vm.currentQuestion = questionID
	vm.voters = make(map[string]string)
	vm.votingActive = true
	vm.timerDuration = duration
	vm.onVoteComplete = onComplete

	vm.votes[questionID] = make(map[string]int)
	for _, choice := range choiceIDs {
		vm.votes[questionID][choice] = 0
	}

	if vm.timer != nil {
		vm.timer.Stop()
	}

	vm.timer = time.AfterFunc(duration, func() {
		vm.EndVoting()
	})

	payload := map[string]any{
		"question_id": questionID,
		"duration":    duration.Seconds(),
	}

	if question != "" {
		payload["question"] = question
	}

	if len(choiceObjects) > 0 {
		payload["choices"] = choiceObjects
	} else {
		payload["choices"] = choiceIDs
	}

	vm.broadcast <- &Message{
		Type:    "voting_started",
		Payload: payload,
	}
}

// SubmitVote records a vote from a user.
func (vm *VoteManager) SubmitVote(voterID, choiceID string) error {
	vm.mu.Lock()
	defer vm.mu.Unlock()

	if !vm.votingActive {
		return nil
	}

	if previousChoice, hasVoted := vm.voters[voterID]; hasVoted {
		if vm.votes[vm.currentQuestion] != nil {
			vm.votes[vm.currentQuestion][previousChoice]--
		}
	}

	vm.voters[voterID] = choiceID
	if vm.votes[vm.currentQuestion] == nil {
		vm.votes[vm.currentQuestion] = make(map[string]int)
	}

	vm.votes[vm.currentQuestion][choiceID]++

	vm.broadcastResults()

	return nil
}

// EndVoting stops the current voting session and determines the winner.
func (vm *VoteManager) EndVoting() {
	vm.mu.Lock()
	defer vm.mu.Unlock()

	if !vm.votingActive {
		return
	}

	vm.votingActive = false

	if vm.timer != nil {
		vm.timer.Stop()
	}

	results := vm.votes[vm.currentQuestion]
	winner := vm.determineWinner(results)

	vm.broadcast <- &Message{
		Type: "voting_ended",
		Payload: map[string]any{
			"question_id": vm.currentQuestion,
			"results":     results,
			"winner":      winner,
		},
	}

	if vm.onVoteComplete != nil {
		go vm.onVoteComplete(results, winner)
	}
}

// determineWinner finds the choice with the most votes.
func (vm *VoteManager) determineWinner(results map[string]int) string {
	maxVotes := 0
	winner := ""

	for choiceID, count := range results {
		if count > maxVotes {
			maxVotes = count
			winner = choiceID
		}
	}

	return winner
}

// broadcastResults sends current vote counts to all clients.
func (vm *VoteManager) broadcastResults() {
	results := make(map[string]int)

	if vm.votes[vm.currentQuestion] != nil {
		maps.Copy(results, vm.votes[vm.currentQuestion])
	}

	vm.broadcast <- &Message{
		Type: "vote_update",
		Payload: map[string]any{
			"question_id": vm.currentQuestion,
			"results":     results,
			"total":       len(vm.voters),
		},
	}
}

// sendState sends the current voting state to a specific client.
func (vm *VoteManager) sendState(client *websocket.Conn) {
	vm.mu.RLock()
	defer vm.mu.RUnlock()

	state := map[string]any{
		"voting_active": vm.votingActive,
		"question_id":   vm.currentQuestion,
	}

	if vm.votingActive && vm.votes[vm.currentQuestion] != nil {
		state["results"] = vm.votes[vm.currentQuestion]
		state["total"] = len(vm.voters)
	}

	message := &Message{
		Type:    "state",
		Payload: state,
	}

	err := client.WriteJSON(message)
	if err != nil {
		log.Printf("Error sending state to client: %v", err)
	}
}

// GetResults returns the current vote counts.
func (vm *VoteManager) GetResults(questionID string) map[string]int {
	vm.mu.RLock()
	defer vm.mu.RUnlock()

	results := make(map[string]int)

	if vm.votes[questionID] != nil {
		maps.Copy(results, vm.votes[questionID])
	}

	return results
}

// RegisterClient adds a WebSocket client.
func (vm *VoteManager) RegisterClient(conn *websocket.Conn) {
	vm.register <- conn
}

// UnregisterClient removes a WebSocket client.
func (vm *VoteManager) UnregisterClient(conn *websocket.Conn) {
	vm.unregister <- conn
}

// BroadcastMessage sends a custom message to all clients.
func (vm *VoteManager) BroadcastMessage(msgType string, payload map[string]any) {
	vm.broadcast <- &Message{
		Type:    msgType,
		Payload: payload,
	}
}

// IsVotingActive returns whether voting is currently active.
func (vm *VoteManager) IsVotingActive() bool {
	vm.mu.RLock()
	defer vm.mu.RUnlock()

	return vm.votingActive
}

// VoteMessage represents an incoming vote.
type VoteMessage struct {
	Type     string `json:"type"`
	VoterID  string `json:"voter_id"`
	ChoiceID string `json:"choice_id"`
}

// HandleVoteMessage processes incoming vote messages.
func (vm *VoteManager) HandleVoteMessage(data []byte) error {
	var msg VoteMessage
	if err := json.Unmarshal(data, &msg); err != nil {
		return err
	}

	if msg.Type == "vote" {
		return vm.SubmitVote(msg.VoterID, msg.ChoiceID)
	}

	return nil
}

// ResetVoting clears all voting state.
func (vm *VoteManager) ResetVoting() {
	vm.mu.Lock()
	defer vm.mu.Unlock()

	if vm.timer != nil {
		vm.timer.Stop()
		vm.timer = nil
	}

	vm.votingActive = false
	vm.currentQuestion = ""
	vm.voters = make(map[string]string)
	// clear the history
	vm.votes = make(map[string]map[string]int)
	vm.onVoteComplete = nil

	vm.broadcast <- &Message{
		Type: "voting_reset",
		Payload: map[string]any{
			"status": "reset",
		},
	}
}

// ClearQuestionVotes clears votes for a specific question only.
func (vm *VoteManager) ClearQuestionVotes(questionID string) {
	vm.mu.Lock()
	defer vm.mu.Unlock()

	if vm.timer != nil {
		vm.timer.Stop()
		vm.timer = nil
	}

	vm.votingActive = false
	vm.currentQuestion = ""
	vm.voters = make(map[string]string)

	if questionID != "" {
		delete(vm.votes, questionID)
	}

	vm.onVoteComplete = nil

	vm.broadcast <- &Message{
		Type: "voting_reset",
		Payload: map[string]any{
			"status": "reset",
		},
	}
}
